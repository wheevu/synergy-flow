package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/synergyflow/backend/internal/middleware"
)

type Config struct{ DatabaseURL, RedisURL, JWTSecret, FrontendURL, S3Bucket, S3Region, S3Endpoint, S3AccessKey, S3SecretKey, ResendAPIKey, FromEmail string }
type Server struct {
	cfg     Config
	db      *pgxpool.Pool
	redis   *redis.Client
	s3      *s3.Client
	presign *s3.PresignClient
	router  *gin.Engine
}
type User struct{ ID, Name, Email string }
type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}
type Event struct {
	Type      string `json:"type,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	ActorID   string `json:"actorId,omitempty"`
	Data      any    `json:"data"`
}

func LoadConfig() Config {
	cfg := Config{
		DatabaseURL: env("DATABASE_URL", "postgres://synergy:synergy@localhost:5432/synergyflow?sslmode=disable"), RedisURL: env("REDIS_URL", "redis://localhost:6379/0"), JWTSecret: env("JWT_SECRET", "dev-only-change-me"), FrontendURL: env("FRONTEND_URL", "http://localhost:5173"),
		S3Bucket: env("S3_BUCKET", "synergyflow-dev"), S3Region: env("AWS_REGION", "us-east-1"), S3Endpoint: os.Getenv("S3_ENDPOINT"), S3AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"), S3SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), ResendAPIKey: os.Getenv("RESEND_API_KEY"), FromEmail: env("FROM_EMAIL", "SynergyFlow <noreply@example.com>"),
	}
	if cfg.JWTSecret == "dev-only-change-me" || cfg.JWTSecret == "" {
		log.Println("[WARN] JWT_SECRET is using the default dev value. Generate a strong secret with: openssl rand -base64 32")
	}
	return cfg
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	ro, _ := redis.ParseURL(cfg.RedisURL)
	rdb := redis.NewClient(ro)
	awsOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.S3Region)}
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		awsOpts = append(awsOpts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return nil, err
	}
	s3c := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true
		}
	})
	s := &Server{cfg: cfg, db: db, redis: rdb, s3: s3c, presign: s3.NewPresignClient(s3c)}
	s.routes()
	return s, nil
}
func (s *Server) Run(port string) error {
	if port == "" {
		port = "8080"
	}
	return s.router.Run(":" + port)
}

func (s *Server) routes() {
	r := gin.New()
	// Global middleware (no timeout — applied selectively below).
	r.Use(middleware.RequestID(), middleware.SecurityHeaders(), middleware.StructuredLogger(), gin.Recovery(), middleware.RequestSizeLimit(12<<20), cors.New(cors.Config{AllowOrigins: []string{s.cfg.FrontendURL}, AllowHeaders: []string{"Authorization", "Content-Type"}, AllowMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}, AllowCredentials: true}))
	s.router = r

	// Health & readiness — no auth, no long timeout needed but also safe with it.
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true, "service": "synergyflow-api"}) })
	r.GET("/ready", func(c *gin.Context) {
		dbOK := s.db.Ping(c) == nil
		redisOK := s.redis.Ping(c).Err() == nil
		if !dbOK {
			c.JSON(503, gin.H{"ok": false, "error": "database unavailable"})
			return
		}
		if !redisOK {
			c.JSON(503, gin.H{"ok": false, "error": "redis unavailable"})
			return
		}
		c.JSON(200, gin.H{"ok": true, "database": "ok", "redis": "ok"})
	})

	// API group — most routes get a 30-second timeout.
	a := r.Group("/api")
	a.Use(middleware.TimeoutMiddleware(30 * time.Second))

	// Unauthenticated auth routes
	a.POST("/auth/register", s.register)
	a.POST("/auth/login", s.login)
	a.POST("/auth/refresh", s.refresh)
	a.POST("/auth/logout", s.auth(), s.logout)
	a.GET("/invites/:token", s.getInvite)
	a.POST("/invites/:token/accept", s.auth(), s.acceptInvite)

	// Authenticated routes (non-SSE)
	p := a.Group("")
	p.Use(s.auth())
	p.GET("/me", s.me)
	p.GET("/workspaces", s.listWorkspaces)
	p.POST("/workspaces", s.createWorkspace)
	p.GET("/workspaces/:id", s.getWorkspace)
	p.GET("/workspaces/:id/members", s.listMembers)
	p.DELETE("/workspaces/:id/members/:uid", s.removeMember)
	p.PATCH("/workspaces/:id/members/:uid", s.updateMemberRole)
	p.POST("/workspaces/:id/invites", s.createInvite)
	p.GET("/workspaces/:id/invites", s.listInvites)
	p.GET("/workspaces/:id/activity", s.workspaceActivity)
	p.GET("/workspaces/:id/dashboard", s.dashboard)
	p.GET("/workspaces/:id/projects", s.listProjects)
	p.POST("/workspaces/:id/projects", s.createProject)
	p.PATCH("/projects/:id", s.updateProject)
	p.DELETE("/projects/:id", s.deleteProject)
	p.GET("/projects/:id/board", s.getBoard)
	p.GET("/projects/:id/tasks", s.searchTasks)
	p.POST("/projects/:id/tasks", s.createTask)
	p.GET("/tasks/:id", s.getTask)
	p.PATCH("/tasks/:id", s.updateTask)
	p.DELETE("/tasks/:id", s.deleteTask)
	p.POST("/tasks/:id/move", s.moveTask)
	p.GET("/tasks/:id/comments", s.comments)
	p.POST("/tasks/:id/comments", s.createComment)
	p.GET("/tasks/:id/attachments", s.listAttachments)
	p.POST("/tasks/:id/attachments", s.uploadAttachment)
	p.GET("/attachments/:id", s.getAttachment)
	p.DELETE("/attachments/:id", s.deleteAttachment)
	p.GET("/notifications", s.notifications)
	p.POST("/notifications/read", s.markNotificationsRead)
	p.POST("/projects/:id/ai/analyze", s.aiAnalyze)

	// SSE endpoint — registered directly on the api group WITHOUT timeout middleware
	// so the long-lived connection is not interrupted.
	a.GET("/projects/:id/events", s.auth(), s.events)
}

func (s *Server) register(c *gin.Context) {
	var in struct{ Name, Email, Password string }
	if bind(c, &in) {
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	var u User
	err := s.db.QueryRow(c, "insert into users(name,email,password_hash) values($1,$2,$3) returning id,name,email", in.Name, strings.ToLower(in.Email), string(hash)).Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		fail(c, 409, err)
		return
	}
	toks, _ := s.issueTokens(c, u.ID)
	c.JSON(201, gin.H{"user": u, "tokens": toks})
}
func (s *Server) login(c *gin.Context) {
	var in struct{ Email, Password string }
	if bind(c, &in) {
		return
	}
	var id, name, email, hash string
	err := s.db.QueryRow(c, "select id,name,email,password_hash from users where email=$1", strings.ToLower(in.Email)).Scan(&id, &name, &email, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}
	toks, _ := s.issueTokens(c, id)
	c.JSON(200, gin.H{"user": User{id, name, email}, "tokens": toks})
}
func (s *Server) refresh(c *gin.Context) {
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	if bind(c, &in) {
		return
	}
	h := hashToken(in.RefreshToken)
	var sid, uid string
	err := s.db.QueryRow(c, "select id,user_id from sessions where refresh_token_hash=$1 and revoked_at is null and expires_at>now()", h).Scan(&sid, &uid)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid refresh token"})
		return
	}
	s.db.Exec(c, "update sessions set revoked_at=now() where id=$1", sid)
	toks, _ := s.issueTokens(c, uid)
	c.JSON(200, gin.H{"tokens": toks})
}
func (s *Server) logout(c *gin.Context) {
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = c.ShouldBindJSON(&in)
	if in.RefreshToken != "" {
		s.db.Exec(c, "update sessions set revoked_at=now() where refresh_token_hash=$1", hashToken(in.RefreshToken))
	}
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) me(c *gin.Context) { u := mustUser(c); c.JSON(200, gin.H{"user": u}) }

func (s *Server) listWorkspaces(c *gin.Context) {
	rows, err := s.db.Query(c, "select w.id,w.name,w.slug,wm.role from workspaces w join workspace_members wm on wm.workspace_id=w.id where wm.user_id=$1 order by w.created_at", userID(c))
	scanRows(c, rows, err)
}
func (s *Server) createWorkspace(c *gin.Context) {
	var in struct{ Name string }
	if bind(c, &in) {
		return
	}
	slug := slugify(in.Name) + "-" + randString(4)
	tx, err := s.db.Begin(c)
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer tx.Rollback(c)
	var id string
	err = tx.QueryRow(c, "insert into workspaces(name,slug,created_by) values($1,$2,$3) returning id", in.Name, slug, userID(c)).Scan(&id)
	if err == nil {
		_, err = tx.Exec(c, "insert into workspace_members(workspace_id,user_id,role) values($1,$2,'Owner')", id, userID(c))
	}
	if err != nil {
		fail(c, 400, err)
		return
	}
	tx.Commit(c)
	c.JSON(201, gin.H{"id": id, "name": in.Name, "slug": slug, "role": "Owner"})
}
func (s *Server) getWorkspace(c *gin.Context) {
	if !s.can(c, c.Param("id"), "Viewer") {
		return
	}
	row := s.db.QueryRow(c, "select id,name,slug from workspaces where id=$1", c.Param("id"))
	var id, n, slug string
	if err := row.Scan(&id, &n, &slug); err != nil {
		fail(c, 404, err)
		return
	}
	c.JSON(200, gin.H{"id": id, "name": n, "slug": slug})
}
func (s *Server) listMembers(c *gin.Context) {
	if !s.can(c, c.Param("id"), "Viewer") {
		return
	}
	// Cast u.email::text to avoid pgx type-registration issues with the citext extension OID.
	rows, err := s.db.Query(c, "select u.id,u.name,u.email::text,wm.role,wm.joined_at from workspace_members wm join users u on u.id=wm.user_id where workspace_id=$1 order by wm.joined_at", c.Param("id"))
	scanRows(c, rows, err)
}

func (s *Server) removeMember(c *gin.Context) {
	wid := c.Param("id")
	if !s.can(c, wid, "Admin") {
		return
	}
	if c.Param("uid") == userID(c) {
		c.JSON(400, gin.H{"error": "cannot remove yourself"})
		return
	}
	_, err := s.db.Exec(c, "delete from workspace_members where workspace_id=$1 and user_id=$2 and role<>'Owner'", wid, c.Param("uid"))
	if err != nil {
		fail(c, 400, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) updateMemberRole(c *gin.Context) {
	wid := c.Param("id")
	if !s.can(c, wid, "Admin") {
		return
	}
	var in struct{ Role string }
	if bind(c, &in) {
		return
	}
	if in.Role != "Viewer" && in.Role != "Member" && in.Role != "Admin" {
		c.JSON(400, gin.H{"error": "invalid role"})
		return
	}
	_, err := s.db.Exec(c, "update workspace_members set role=$3 where workspace_id=$1 and user_id=$2 and role<>'Owner'", wid, c.Param("uid"), in.Role)
	if err != nil {
		fail(c, 400, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) listInvites(c *gin.Context) {
	if !s.can(c, c.Param("id"), "Admin") {
		return
	}
	// Cast email::text to avoid pgx type-registration issues with citext OID.
	rows, err := s.db.Query(c, "select id,email::text,role,expires_at from workspace_invites where workspace_id=$1 and accepted_at is null and expires_at>now() order by created_at desc", c.Param("id"))
	scanRows(c, rows, err)
}

func (s *Server) createInvite(c *gin.Context) {
	wid := c.Param("id")
	if !s.can(c, wid, "Admin") {
		return
	}
	var in struct{ Email, Role string }
	if bind(c, &in) {
		return
	}
	if in.Role == "" {
		in.Role = "Member"
	}
	token := randString(32)
	_, err := s.db.Exec(c, "insert into workspace_invites(workspace_id,email,role,token,created_by,expires_at) values($1,$2,$3,$4,$5,now()+interval '7 days')", wid, strings.ToLower(in.Email), in.Role, token, userID(c))
	if err != nil {
		fail(c, 400, err)
		return
	}
	s.queueEmail(c, in.Email, "You're invited to SynergyFlow", fmt.Sprintf("Open %s/invite/%s to join the workspace.", s.cfg.FrontendURL, token))
	c.JSON(201, gin.H{"token": token, "url": s.cfg.FrontendURL + "/invite/" + token})
}
func (s *Server) getInvite(c *gin.Context) {
	var email, role, wid, wname string
	var expired, accepted bool
	err := s.db.QueryRow(c, "select i.email,i.role,w.id,w.name,i.expires_at<=now(),i.accepted_at is not null from workspace_invites i join workspaces w on w.id=i.workspace_id where token=$1", c.Param("token")).Scan(&email, &role, &wid, &wname, &expired, &accepted)
	if err != nil {
		fail(c, 404, err)
		return
	}
	if accepted {
		c.JSON(409, gin.H{"error": "invite already accepted"})
		return
	}
	if expired {
		c.JSON(410, gin.H{"error": "invite expired"})
		return
	}
	c.JSON(200, gin.H{"email": email, "role": role, "workspaceId": wid, "workspaceName": wname})
}
func (s *Server) acceptInvite(c *gin.Context) {
	tx, _ := s.db.Begin(c)
	defer tx.Rollback(c)
	var wid, role string
	err := tx.QueryRow(c, "select workspace_id,role from workspace_invites where token=$1 and accepted_at is null and expires_at>now() for update", c.Param("token")).Scan(&wid, &role)
	if err != nil {
		fail(c, 404, err)
		return
	}
	_, err = tx.Exec(c, "insert into workspace_members(workspace_id,user_id,role) values($1,$2,$3) on conflict do nothing", wid, userID(c), role)
	if err == nil {
		_, err = tx.Exec(c, "update workspace_invites set accepted_at=now() where token=$1", c.Param("token"))
	}
	if err != nil {
		fail(c, 400, err)
		return
	}
	tx.Commit(c)
	s.activity(c, wid, nil, "member.joined", gin.H{"userId": userID(c)})
	c.JSON(200, gin.H{"workspaceId": wid})
}

func (s *Server) listProjects(c *gin.Context) {
	if !s.can(c, c.Param("id"), "Viewer") {
		return
	}
	rows, err := s.db.Query(c, "select id,name,description from projects where workspace_id=$1 order by created_at", c.Param("id"))
	scanRows(c, rows, err)
}
func (s *Server) createProject(c *gin.Context) {
	wid := c.Param("id")
	if !s.can(c, wid, "Member") {
		return
	}
	var in struct{ Name, Description string }
	if bind(c, &in) {
		return
	}
	tx, _ := s.db.Begin(c)
	defer tx.Rollback(c)
	var pid string
	err := tx.QueryRow(c, "insert into projects(workspace_id,name,description,created_by) values($1,$2,$3,$4) returning id", wid, in.Name, in.Description, userID(c)).Scan(&pid)
	cols := []string{"Backlog", "Todo", "In Progress", "In Review", "Done"}
	for i, n := range cols {
		if err == nil {
			_, err = tx.Exec(c, "insert into project_columns(project_id,name,position) values($1,$2,$3)", pid, n, i)
		}
	}
	if err != nil {
		fail(c, 400, err)
		return
	}
	tx.Commit(c)
	c.JSON(201, gin.H{"id": pid, "name": in.Name, "description": in.Description})
}
func (s *Server) updateProject(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Admin") {
		return
	}
	var in struct{ Name, Description string }
	if bind(c, &in) {
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		c.JSON(400, gin.H{"error": "project name is required"})
		return
	}
	_, err := s.db.Exec(c, "update projects set name=$2, description=$3, updated_at=now() where id=$1", pid, strings.TrimSpace(in.Name), in.Description)
	if err != nil {
		fail(c, 400, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) deleteProject(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Owner") {
		return
	}
	_, err := s.db.Exec(c, "delete from projects where id=$1", pid)
	if err != nil {
		fail(c, 400, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) getBoard(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Viewer") {
		return
	}
	rows, err := s.db.Query(c, `select c.id,c.name,c.position,coalesce(json_agg(json_build_object('id',t.id,'title',t.title,'description',t.description,'priority',t.priority,'assigneeId',t.assignee_id,'dueDate',t.due_date,'labels',t.labels,'position',t.position) order by t.position) filter (where t.id is not null),'[]') from project_columns c left join tasks t on t.column_id=c.id where c.project_id=$1 group by c.id order by c.position`, pid)
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var id, n string
		var pos int
		var tasks []byte
		rows.Scan(&id, &n, &pos, &tasks)
		var arr any
		json.Unmarshal(tasks, &arr)
		out = append(out, gin.H{"id": id, "name": n, "position": pos, "tasks": arr})
	}
	c.JSON(200, gin.H{"columns": out})
}
func (s *Server) createTask(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Member") {
		return
	}
	var in struct {
		Title, Description, ColumnID, Priority, AssigneeID string
		DueDate                                            *time.Time
		Labels                                             []string
	}
	if bind(c, &in) {
		return
	}
	if in.Priority == "" {
		in.Priority = "Medium"
	}
	if in.ColumnID == "" {
		_ = s.db.QueryRow(c, "select id from project_columns where project_id=$1 order by position limit 1", pid).Scan(&in.ColumnID)
	}
	var id string
	err := s.db.QueryRow(c, "insert into tasks(project_id,column_id,title,description,priority,assignee_id,due_date,labels,created_by,position) values($1,$2,$3,$4,$5,nullif($6,'')::uuid,$7,$8,$9,coalesce((select max(position)+1 from tasks where column_id=$2),0)) returning id", pid, in.ColumnID, in.Title, in.Description, in.Priority, in.AssigneeID, in.DueDate, in.Labels, userID(c)).Scan(&id)
	if err != nil {
		fail(c, 400, err)
		return
	}
	if in.AssigneeID != "" {
		s.notifyRef(c, in.AssigneeID, "task.assigned", "Task assigned", "You were assigned to "+in.Title, "task", id)
	}
	s.projectEvent(c, pid, "task.created", gin.H{"id": id})
	c.JSON(201, gin.H{"id": id})
}
func (s *Server) getTask(c *gin.Context) {
	rows, err := s.db.Query(c, "select t.id,t.title,t.description,t.priority,t.assignee_id,t.due_date,t.labels,t.created_at,t.updated_at,c.name as status from tasks t join project_columns c on c.id=t.column_id where t.id=$1", c.Param("id"))
	scanRows(c, rows, err)
}
func (s *Server) updateTask(c *gin.Context) {
	var in map[string]any
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	if !s.canTask(c, c.Param("id"), "Member") {
		return
	}
	var due *time.Time
	clearDue := false
	if raw, ok := in["dueDate"]; ok {
		s := strings.TrimSpace(fmt.Sprint(raw))
		if s == "" || s == "<nil>" || s == "null" {
			clearDue = true
		} else if parsed, err := time.Parse("2006-01-02", s); err == nil {
			due = &parsed
		} else if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			due = &parsed
		} else {
			c.JSON(400, gin.H{"error": "invalid dueDate"})
			return
		}
	}
	var dueArg any
	if clearDue {
		dueArg = nil
	} else if due != nil {
		dueArg = *due
	}
	_, err := s.db.Exec(c, "update tasks set title=coalesce($2,title), description=coalesce($3,description), priority=coalesce($4,priority), assignee_id=coalesce(nullif($5,'')::uuid,assignee_id), labels=coalesce($6,labels), due_date=case when $7 then null when $8::timestamptz is not null then $8 else due_date end, updated_at=now() where id=$1", c.Param("id"), strp(in, "title"), strp(in, "description"), strp(in, "priority"), strp(in, "assigneeId"), stringSlice(in["labels"]), clearDue, dueArg)
	if err != nil {
		fail(c, 400, err)
		return
	}
	pid := s.projectIDForTask(c, c.Param("id"))
	if assignee := strp(in, "assigneeId"); assignee != nil && *assignee != "" {
		s.notifyRef(c, *assignee, "task.assigned", "Task assigned", "A task was assigned to you", "task", c.Param("id"))
	}
	s.projectEvent(c, pid, "task.updated", gin.H{"id": c.Param("id")})
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) deleteTask(c *gin.Context) {
	if !s.canTask(c, c.Param("id"), "Member") {
		return
	}
	pid := s.projectIDForTask(c, c.Param("id"))
	var col string
	var pos int
	_ = s.db.QueryRow(c, "select column_id,position from tasks where id=$1", c.Param("id")).Scan(&col, &pos)
	_, err := s.db.Exec(c, "delete from tasks where id=$1", c.Param("id"))
	if err != nil {
		fail(c, 400, err)
		return
	}
	if col != "" {
		_, _ = s.db.Exec(c, "update tasks set position=position-1 where column_id=$1 and position>$2", col, pos)
	}
	s.projectEvent(c, pid, "task.deleted", gin.H{"id": c.Param("id")})
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) moveTask(c *gin.Context) {
	var in struct {
		ColumnID string `json:"columnId"`
		Position int    `json:"position"`
	}
	if bind(c, &in) {
		return
	}
	tid := c.Param("id")
	if !s.canTask(c, tid, "Member") {
		return
	}
	tx, err := s.db.Begin(c)
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer tx.Rollback(c)
	var oldCol, pid string
	var oldPos int
	err = tx.QueryRow(c, "select column_id,project_id,position from tasks where id=$1 for update", tid).Scan(&oldCol, &pid, &oldPos)
	if err != nil {
		fail(c, 404, err)
		return
	}
	var destProject string
	if err = tx.QueryRow(c, "select project_id from project_columns where id=$1", in.ColumnID).Scan(&destProject); err != nil || destProject != pid {
		c.JSON(400, gin.H{"error": "invalid destination column"})
		return
	}
	if _, err = tx.Exec(c, "select id from tasks where column_id in ($1,$2) order by column_id, position for update", oldCol, in.ColumnID); err != nil {
		fail(c, 400, err)
		return
	}
	var destCount int
	if err = tx.QueryRow(c, "select count(*) from tasks where column_id=$1", in.ColumnID).Scan(&destCount); err != nil {
		fail(c, 400, err)
		return
	}
	maxPosition := destCount
	if oldCol == in.ColumnID && maxPosition > 0 {
		maxPosition--
	}
	if in.Position < 0 {
		in.Position = 0
	}
	if in.Position > maxPosition {
		in.Position = maxPosition
	}
	if oldCol == in.ColumnID && oldPos == in.Position {
		if err = tx.Commit(c); err != nil {
			fail(c, 500, err)
			return
		}
		c.JSON(200, gin.H{"ok": true})
		return
	}
	// Transactional movement keeps a dense integer order per column. We first close the gap in
	// the source column, then open a gap in the validated destination column, and finally place
	// the task at a clamped index so clients cannot create sparse or cross-project ordering.
	_, err = tx.Exec(c, "update tasks set position=position-1 where column_id=$1 and position>$2", oldCol, oldPos)
	if err == nil {
		_, err = tx.Exec(c, "update tasks set position=position+1 where column_id=$1 and position>=$2", in.ColumnID, in.Position)
	}
	if err == nil {
		_, err = tx.Exec(c, "update tasks set column_id=$2,position=$3,updated_at=now() where id=$1", tid, in.ColumnID, in.Position)
	}
	if err != nil {
		fail(c, 400, err)
		return
	}
	if err = tx.Commit(c); err != nil {
		fail(c, 500, err)
		return
	}
	s.projectEvent(c, pid, "task.moved", gin.H{"id": tid, "columnId": in.ColumnID, "position": in.Position})
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) searchTasks(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Viewer") {
		return
	}
	q := "%" + c.Query("q") + "%"
	rows, err := s.db.Query(c, `select t.id,t.title,t.priority,t.assignee_id,t.due_date,t.labels,c.name status from tasks t join project_columns c on c.id=t.column_id where t.project_id=$1 and ($2='%%' or t.search_vector @@ plainto_tsquery('english',$3) or t.title ilike $2 or t.description ilike $2) and ($4='' or t.priority=$4) and ($5='' or c.id=$5) order by t.updated_at desc limit 100`, pid, q, c.Query("q"), c.Query("priority"), c.Query("status"))
	scanRows(c, rows, err)
}

func (s *Server) comments(c *gin.Context) {
	if !s.canTask(c, c.Param("id"), "Viewer") {
		return
	}
	rows, err := s.db.Query(c, "select tc.id,tc.body,u.name author,tc.created_at from task_comments tc join users u on u.id=tc.author_id where task_id=$1 order by tc.created_at", c.Param("id"))
	scanRows(c, rows, err)
}
func (s *Server) createComment(c *gin.Context) {
	if !s.canTask(c, c.Param("id"), "Member") {
		return
	}
	var in struct{ Body string }
	if bind(c, &in) {
		return
	}
	var id string
	err := s.db.QueryRow(c, "insert into task_comments(task_id,author_id,body) values($1,$2,$3) returning id", c.Param("id"), userID(c), in.Body).Scan(&id)
	if err != nil {
		fail(c, 400, err)
		return
	}
	pid := s.projectIDForTask(c, c.Param("id"))
	var assignee string
	_ = s.db.QueryRow(c, "select coalesce(assignee_id::text,'') from tasks where id=$1", c.Param("id")).Scan(&assignee)
	if assignee != "" && assignee != userID(c) {
		s.notifyRef(c, assignee, "comment.created", "New comment", "Someone commented on a task assigned to you", "task", c.Param("id"))
	}
	s.projectEvent(c, pid, "comment.created", gin.H{"id": id, "taskId": c.Param("id")})
	c.JSON(201, gin.H{"id": id})
}

func (s *Server) listAttachments(c *gin.Context) {
	if !s.canTask(c, c.Param("id"), "Viewer") {
		return
	}
	rows, err := s.db.Query(c, "select ta.id,ta.file_name as name,ta.content_type as type,ta.size_bytes as size,ta.created_at as time,u.name as uploader from task_attachments ta join users u on u.id=ta.uploader_id where ta.task_id=$1 order by ta.created_at", c.Param("id"))
	scanRows(c, rows, err)
}

func (s *Server) deleteAttachment(c *gin.Context) {
	// Simple authorization: if user can edit the task they can delete its attachments
	var tid string
	err := s.db.QueryRow(c, "select task_id from task_attachments where id=$1", c.Param("id")).Scan(&tid)
	if err != nil {
		fail(c, 404, err)
		return
	}
	if !s.canTask(c, tid, "Member") {
		return
	}
	_, err = s.db.Exec(c, "delete from task_attachments where id=$1", c.Param("id"))
	if err != nil {
		fail(c, 400, err)
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) getAttachment(c *gin.Context) {
	var tid, key string
	err := s.db.QueryRow(c, "select task_id,storage_key from task_attachments where id=$1", c.Param("id")).Scan(&tid, &key)
	if err != nil {
		fail(c, 404, err)
		return
	}
	if !s.canTask(c, tid, "Viewer") {
		return
	}
	url, err := s.presign.PresignGetObject(c, &s3.GetObjectInput{Bucket: aws.String(s.cfg.S3Bucket), Key: aws.String(key)})
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(200, gin.H{"url": url.URL})
}

func (s *Server) uploadAttachment(c *gin.Context) {
	if !s.canTask(c, c.Param("id"), "Member") {
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		fail(c, 400, err)
		return
	}
	if f.Size > 10<<20 {
		c.JSON(400, gin.H{"error": "file too large"})
		return
	}
	key, err := s.putObject(c, f)
	if err != nil {
		fail(c, 500, err)
		return
	}
	var id string
	err = s.db.QueryRow(c, "insert into task_attachments(task_id,uploader_id,file_name,content_type,size_bytes,storage_key) values($1,$2,$3,$4,$5,$6) returning id", c.Param("id"), userID(c), f.Filename, f.Header.Get("Content-Type"), f.Size, key).Scan(&id)
	if err != nil {
		fail(c, 400, err)
		return
	}
	s.projectEvent(c, s.projectIDForTask(c, c.Param("id")), "attachment.created", gin.H{"id": id, "taskId": c.Param("id")})
	c.JSON(201, gin.H{"id": id, "fileName": f.Filename, "storageKey": key})
}

func (s *Server) events(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Viewer") {
		return
	}
	channel := "project:" + pid
	sub := s.redis.Subscribe(c, channel)
	defer sub.Close()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg := <-sub.Channel():
			c.SSEvent("message", msg.Payload)
			return true
		case <-time.After(25 * time.Second):
			c.SSEvent("ping", gin.H{"ts": time.Now()})
			return true
		}
	})
}
func (s *Server) workspaceActivity(c *gin.Context) {
	if !s.can(c, c.Param("id"), "Viewer") {
		return
	}
	rows, err := s.db.Query(c, "select id,event_type,metadata,created_at from activity_events where workspace_id=$1 order by created_at desc limit 50", c.Param("id"))
	scanRows(c, rows, err)
}
func (s *Server) dashboard(c *gin.Context) {
	if !s.can(c, c.Param("id"), "Viewer") {
		return
	}
	row := s.db.QueryRow(c, `select count(distinct t.id) filter(where pc.name='Done'), count(distinct t.id) filter(where t.due_date<now() and pc.name<>'Done'), count(distinct wm.user_id) from workspaces w join workspace_members wm on wm.workspace_id=w.id left join projects p on p.workspace_id=w.id left join project_columns pc on pc.project_id=p.id left join tasks t on t.column_id=pc.id where w.id=$1`, c.Param("id"))
	var done, over, members int
	_ = row.Scan(&done, &over, &members)
	c.JSON(200, gin.H{"tasksCompleted": done, "overdueTasks": over, "activeMembers": members})
}
func (s *Server) notifications(c *gin.Context) {
	rows, err := s.db.Query(c, "select id,type,title,body,read_at,created_at,resource_type,resource_id from notifications where user_id=$1 order by created_at desc limit 50", userID(c))
	scanRows(c, rows, err)
}
func (s *Server) markNotificationsRead(c *gin.Context) {
	s.db.Exec(c, "update notifications set read_at=now() where user_id=$1 and read_at is null", userID(c))
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) aiAnalyze(c *gin.Context) {
	pid := c.Param("id")
	if !s.canProject(c, pid, "Viewer") {
		return
	}
	var in struct{ Prompt string `json:"prompt"` }
	if bind(c, &in) {
		return
	}

	// Fetch workspace_id for this project to load members and activity
	var wid string
	_ = s.db.QueryRow(c, "select workspace_id from projects where id=$1", pid).Scan(&wid)

	// Fetch tasks with column names for the project
	type aiTask struct {
		ID, Title, Description, Priority, Status string
		AssigneeID, DueDate, Labels               *string
		UpdatedAt                                time.Time
	}
	var tasks []aiTask
	rows, err := s.db.Query(c, `select t.id,t.title,t.description,t.priority,c.name,t.assignee_id::text,t.due_date::date::text,coalesce(t.labels::text,''),t.updated_at from tasks t join project_columns c on c.id=t.column_id where t.project_id=$1 order by t.updated_at desc`, pid)
	if err != nil {
		log.Printf("aiAnalyze query error: %v", err)
	} else {
		for rows.Next() {
			var t aiTask
			var desc []byte
			if err := rows.Scan(&t.ID, &t.Title, &desc, &t.Priority, &t.Status, &t.AssigneeID, &t.DueDate, &t.Labels, &t.UpdatedAt); err != nil {
				log.Printf("aiAnalyze scan error: %v", err)
				continue
			}
			if len(desc) > 0 {
				t.Description = string(desc)
			}
			tasks = append(tasks, t)
		}
		if err := rows.Err(); err != nil {
			log.Printf("aiAnalyze rows error: %v", err)
		}
		rows.Close()
	}

	// Fetch workspace members
	memberNames := map[string]string{}
	if wid != "" {
		mRows, mErr := s.db.Query(c, "select u.id::text,u.name from workspace_members wm join users u on u.id=wm.user_id where wm.workspace_id=$1", wid)
		if mErr == nil {
			for mRows.Next() {
				var uid, name string
				mRows.Scan(&uid, &name)
				memberNames[uid] = name
			}
			mRows.Close()
		}
	}

	// Fetch recent activity for this project
	var recentEvents []string
	if wid != "" {
		aRows, aErr := s.db.Query(c, "select event_type from activity_events where project_id=$1 order by created_at desc limit 10", pid)
		if aErr == nil {
			for aRows.Next() {
				var et string
				aRows.Scan(&et)
				recentEvents = append(recentEvents, et)
			}
			aRows.Close()
		}
	}

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sevenDaysAgo := now.AddDate(0, 0, -7)

	// Analysis metrics + named task lists
	var overdueCount, urgentCount, unassignedCount, highPriorityCount int
	assigneeTaskCount := map[string]int{}
	staleCount := 0
	statusCounts := map[string]int{}
	var urgentOverdueTasks, highOverdueTasks, staleTasks, unassignedTasks, todoHighTasks []aiTask
	for _, t := range tasks {
		statusCounts[t.Status]++
		isOverdue := false
		if t.DueDate != nil && *t.DueDate != "" {
			var d time.Time
			var err error
			if strings.Contains(*t.DueDate, "T") {
				d, err = time.Parse(time.RFC3339, *t.DueDate)
			} else {
				d, err = time.Parse("2006-01-02", *t.DueDate)
			}
			if err == nil && d.Before(startOfToday) {
				overdueCount++
				isOverdue = true
			}
		}
		p := strings.ToLower(t.Priority)
		if p == "urgent" {
			urgentCount++
			if isOverdue {
				urgentOverdueTasks = append(urgentOverdueTasks, t)
			}
		}
		if p == "high" {
			highPriorityCount++
			if isOverdue {
				highOverdueTasks = append(highOverdueTasks, t)
			}
		}
		if t.AssigneeID == nil || *t.AssigneeID == "" {
			unassignedCount++
			unassignedTasks = append(unassignedTasks, t)
		} else {
			assigneeTaskCount[*t.AssigneeID]++
		}
		if t.UpdatedAt.Before(sevenDaysAgo) && t.Status != "Done" {
			staleCount++
			staleTasks = append(staleTasks, t)
		}
		if (p == "high" || p == "urgent") && !strings.EqualFold(t.Status, "Done") && !strings.Contains(strings.ToLower(t.Status), "in progress") {
			todoHighTasks = append(todoHighTasks, t)
		}
	}

	assigneeName := func(uid string) string {
		if n := memberNames[uid]; n != "" {
			return n
		}
		return uid
	}

	var overloaded []string
	for uid, count := range assigneeTaskCount {
		if count >= 5 {
			overloaded = append(overloaded, assigneeName(uid))
		}
	}

	total := len(tasks)
	completed := statusCounts["Done"]
	todoCount := 0
	for sname, cnt := range statusCounts {
		if strings.Contains(strings.ToLower(sname), "todo") || strings.Contains(strings.ToLower(sname), "backlog") {
			todoCount += cnt
		}
	}

	// Helpers for task name lists
	taskNames := func(list []aiTask, limit int) string {
		if len(list) == 0 {
			return ""
		}
		var parts []string
		for i, t := range list {
			if i >= limit {
				break
			}
			parts = append(parts, fmt.Sprintf("'%s'", t.Title))
		}
		return strings.Join(parts, ", ")
	}
	firstTask := func(list []aiTask) string {
		if len(list) == 0 {
			return ""
		}
		return list[0].Title
	}
	assigneeTaskNames := func(uid string, limit int) string {
		var parts []string
		for _, t := range tasks {
			if t.AssigneeID != nil && *t.AssigneeID == uid {
				parts = append(parts, fmt.Sprintf("'%s'", t.Title))
				if len(parts) >= limit {
					break
				}
			}
		}
		return strings.Join(parts, ", ")
	}

	// Build prompt-aware answer
	promptLower := strings.ToLower(in.Prompt)
	var answer string
	var signals []gin.H
	var suggestedActions []string

	// Signals (always returned)
	if overdueCount > 0 {
		sev := "medium"
		if overdueCount >= 3 {
			sev = "high"
		}
		signals = append(signals, gin.H{"label": "Overdue tasks", "value": fmt.Sprintf("%d", overdueCount), "severity": sev})
	}
	if unassignedCount > 0 {
		sev := "low"
		if unassignedCount >= 3 {
			sev = "medium"
		}
		signals = append(signals, gin.H{"label": "Unassigned tasks", "value": fmt.Sprintf("%d", unassignedCount), "severity": sev})
	}
	if urgentCount > 0 {
		signals = append(signals, gin.H{"label": "Urgent priority", "value": fmt.Sprintf("%d", urgentCount), "severity": "high"})
	}
	if len(overloaded) > 0 {
		signals = append(signals, gin.H{"label": "Overloaded assignees", "value": strings.Join(overloaded, ", "), "severity": "medium"})
	}
	if staleCount > 0 {
		sev := "low"
		if staleCount >= 3 {
			sev = "medium"
		}
		signals = append(signals, gin.H{"label": "Stale tasks (7d+)", "value": fmt.Sprintf("%d", staleCount), "severity": sev})
	}
	if completed > 0 && total > 0 {
		pct := (completed * 100) / total
		sev := "low"
		if pct < 20 {
			sev = "medium"
		}
		signals = append(signals, gin.H{"label": "Completion", "value": fmt.Sprintf("%d%% (%d/%d)", pct, completed, total), "severity": sev})
	}

	// Determine answer and suggested actions based on prompt keywords
	switch {
	case strings.Contains(promptLower, "next") || strings.Contains(promptLower, "should we work"):
		var parts []string
		if len(urgentOverdueTasks) > 0 {
			parts = append(parts, fmt.Sprintf("Start with urgent overdue task '%s'.", urgentOverdueTasks[0].Title))
		} else if len(highOverdueTasks) > 0 {
			parts = append(parts, fmt.Sprintf("Start with high overdue task '%s'.", highOverdueTasks[0].Title))
		}
		if len(urgentOverdueTasks) > 1 {
			parts = append(parts, fmt.Sprintf("Also handle '%s'.", urgentOverdueTasks[1].Title))
		} else if len(highOverdueTasks) > 1 {
			parts = append(parts, fmt.Sprintf("Also handle '%s'.", highOverdueTasks[1].Title))
		}
		if len(todoHighTasks) > 0 {
			parts = append(parts, fmt.Sprintf("Then move '%s' into In Progress.", todoHighTasks[0].Title))
		}
		if len(unassignedTasks) > 0 {
			parts = append(parts, fmt.Sprintf("Assign '%s' to unblock the queue.", unassignedTasks[0].Title))
		}
		if len(parts) == 0 {
			parts = append(parts, "The board is looking healthy. Pick the next high-priority task from Todo/Backlog.")
		}
		answer = strings.Join(parts, " ")

	case strings.Contains(promptLower, "risk") || strings.Contains(promptLower, "risky"):
		var parts []string
		if len(urgentOverdueTasks) > 0 {
			parts = append(parts, fmt.Sprintf("'%s' is urgent and overdue.", urgentOverdueTasks[0].Title))
		}
		if len(urgentOverdueTasks) > 1 {
			parts = append(parts, fmt.Sprintf("'%s' is also at risk.", urgentOverdueTasks[1].Title))
		}
		if len(highOverdueTasks) > 0 && len(parts) < 2 {
			parts = append(parts, fmt.Sprintf("'%s' is high priority and overdue.", highOverdueTasks[0].Title))
		}
		if len(parts) == 0 {
			parts = append(parts, fmt.Sprintf("No urgent risks found. %d overdue tasks need review.", overdueCount))
		}
		if len(overloaded) > 0 {
			parts = append(parts, fmt.Sprintf("%s are overloaded with 5+ tasks each.", strings.Join(overloaded, ", ")))
		}
		answer = strings.Join(parts, " ")

	case strings.Contains(promptLower, "overload") || strings.Contains(promptLower, "who looks") || strings.Contains(promptLower, "workload"):
		var parts []string
		if len(overloaded) > 0 {
			for uid, count := range assigneeTaskCount {
				if count >= 5 {
					names := assigneeTaskNames(uid, 3)
					parts = append(parts, fmt.Sprintf("%s has %d tasks, including %s.", assigneeName(uid), count, names))
				}
			}
		}
		if len(parts) == 0 {
			parts = append(parts, "No one is critically overloaded yet. Keep monitoring as backlog grows.")
		}
		if unassignedCount > 0 {
			parts = append(parts, fmt.Sprintf("There are %d unassigned tasks that could be redistributed.", unassignedCount))
		}
		answer = strings.Join(parts, " ")

	case strings.Contains(promptLower, "summarize") || strings.Contains(promptLower, "sprint"):
		answer = fmt.Sprintf("Project has %d tasks total, %d completed, %d overdue, and %d urgent. %d are unassigned and %d are high priority. Completion rate is around %d%%.", total, completed, overdueCount, urgentCount, unassignedCount, highPriorityCount, (completed*100)/max(total, 1))
		if first := firstTask(urgentOverdueTasks); first != "" {
			answer += fmt.Sprintf(" Top risk: '%s' is urgent and overdue.", first)
		} else if first := firstTask(highOverdueTasks); first != "" {
			answer += fmt.Sprintf(" Top risk: '%s' is high priority and overdue.", first)
		}

	case strings.Contains(promptLower, "overdue") || strings.Contains(promptLower, "urgent"):
		var parts []string
		if len(urgentOverdueTasks) > 0 {
			parts = append(parts, fmt.Sprintf("Urgent overdue: %s.", taskNames(urgentOverdueTasks, 3)))
		}
		if len(highOverdueTasks) > 0 {
			parts = append(parts, fmt.Sprintf("High overdue: %s.", taskNames(highOverdueTasks, 3)))
		}
		if len(parts) == 0 {
			parts = append(parts, fmt.Sprintf("There are %d overdue and %d urgent tasks.", overdueCount, urgentCount))
		}
		answer = strings.Join(parts, " ")

	case strings.Contains(promptLower, "change") || strings.Contains(promptLower, "recent") || strings.Contains(promptLower, "what changed"):
		if len(recentEvents) > 0 {
			activitySummary := summarizeEvents(recentEvents)
			answer = fmt.Sprintf("Recent activity: %s.", activitySummary)
		} else {
			answer = "No recent recorded changes. Consider updating task statuses to reflect current progress."
		}

	case strings.Contains(promptLower, "health"):
		answer = fmt.Sprintf("Project health: %d total tasks, %d done (%d%%), %d overdue, %d urgent, %d unassigned. %d task(s) are stale.", total, completed, (completed*100)/max(total, 1), overdueCount, urgentCount, unassignedCount, staleCount)
		if first := firstTask(urgentOverdueTasks); first != "" {
			answer += fmt.Sprintf(" Immediate attention: '%s' is urgent and overdue.", first)
		} else if first := firstTask(highOverdueTasks); first != "" {
			answer += fmt.Sprintf(" Immediate attention: '%s' is high priority and overdue.", first)
		} else if overdueCount > 2 || unassignedCount > 3 {
			answer += " Attention recommended: overdue and unassigned counts are elevated."
		} else {
			answer += " The board looks reasonably healthy."
		}

	case strings.Contains(promptLower, "blocker") || strings.Contains(promptLower, "block"):
		var blockerNamed []string
		for _, t := range tasks {
			if t.Labels != nil && strings.Contains(strings.ToLower(*t.Labels), "block") {
				blockerNamed = append(blockerNamed, fmt.Sprintf("'%s'", t.Title))
			}
		}
		if len(blockerNamed) > 0 {
			answer = fmt.Sprintf("Found blocker-tagged tasks: %s. Resolve these first to unblock downstream work.", strings.Join(blockerNamed, ", "))
		} else {
			var riskiest []string
			for _, t := range urgentOverdueTasks {
				riskiest = append(riskiest, fmt.Sprintf("'%s'", t.Title))
				if len(riskiest) >= 3 {
					break
				}
			}
			if len(riskiest) < 3 {
				for _, t := range highOverdueTasks {
					riskiest = append(riskiest, fmt.Sprintf("'%s'", t.Title))
					if len(riskiest) >= 3 {
						break
					}
				}
			}
			if len(riskiest) < 3 {
				for _, t := range staleTasks {
					riskiest = append(riskiest, fmt.Sprintf("'%s'", t.Title))
					if len(riskiest) >= 3 {
						break
					}
				}
			}
			if len(riskiest) > 0 {
				answer = fmt.Sprintf("No explicit blocker labels. Top risks: %s.", strings.Join(riskiest, ", "))
			} else {
				answer = "No explicit blocker labels found. Monitor overdue and unassigned tasks as implicit blockers."
			}
		}
		if unassignedCount > 0 {
			answer += fmt.Sprintf(" %d unassigned task(s) may also be causing delays.", unassignedCount)
		}

	default:
		answer = fmt.Sprintf("This project has %d tasks: %d completed, %d overdue, %d urgent, and %d unassigned. %d high-priority items are waiting in Todo/Backlog. Ask me about next actions, risks, workload, or recent changes.", total, completed, overdueCount, urgentCount, unassignedCount, highPriorityCount)
	}

	// Build suggested actions from real task data
	suggestedActions = []string{}
	if len(urgentOverdueTasks) > 0 {
		suggestedActions = append(suggestedActions, fmt.Sprintf("Prioritize '%s' — it is urgent and overdue.", urgentOverdueTasks[0].Title))
	}
	if len(highOverdueTasks) > 0 {
		suggestedActions = append(suggestedActions, fmt.Sprintf("Review '%s' — high priority and overdue.", highOverdueTasks[0].Title))
	}
	if len(unassignedTasks) > 0 {
		suggestedActions = append(suggestedActions, fmt.Sprintf("Assign '%s' to a team member.", unassignedTasks[0].Title))
	}
	if len(staleTasks) > 0 {
		suggestedActions = append(suggestedActions, fmt.Sprintf("Update status of '%s' — hasn't moved in 7+ days.", staleTasks[0].Title))
	}
	if len(overloaded) > 0 {
		for uid, count := range assigneeTaskCount {
			if count >= 5 {
				suggestedActions = append(suggestedActions, fmt.Sprintf("Redistribute work from %s (%d tasks).", assigneeName(uid), count))
				break
			}
		}
	}
	if len(todoHighTasks) > 0 {
		suggestedActions = append(suggestedActions, fmt.Sprintf("Move '%s' into In Progress.", todoHighTasks[0].Title))
	}
	if len(suggestedActions) == 0 {
		suggestedActions = append(suggestedActions, "Pick the next high-priority task from Todo/Backlog.")
	}
	if len(suggestedActions) > 5 {
		suggestedActions = suggestedActions[:5]
	}

	c.JSON(200, gin.H{"answer": answer, "signals": signals, "suggestedActions": suggestedActions})
}

func summarizeEvents(events []string) string {
	counts := map[string]int{}
	for _, e := range events {
		counts[e]++
	}
	var parts []string
	for _, label := range []string{"task.created", "task.updated", "task.moved", "comment.created", "attachment.created", "member.joined"} {
		if c := counts[label]; c > 0 {
			verb := map[string]string{
				"task.created":    "created",
				"task.updated":    "updated",
				"task.moved":      "moved",
				"comment.created": "commented",
				"attachment.created": "uploaded attachments",
				"member.joined":   "new members joined",
			}[label]
			if c == 1 {
				parts = append(parts, fmt.Sprintf("1 task %s", verb))
			} else {
				parts = append(parts, fmt.Sprintf("%d %s", c, verb))
			}
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d miscellaneous activity events", len(events))
	}
	return strings.Join(parts, ", ")
}

func (s *Server) auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		tok := strings.TrimPrefix(h, "Bearer ")
		if tok == "" || tok == h {
			tok = c.Query("token")
		}
		claims := &Claims{}
		t, err := jwt.ParseWithClaims(tok, claims, func(*jwt.Token) (any, error) { return []byte(s.cfg.JWTSecret), nil })
		if err != nil || !t.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		var u User
		if err := s.db.QueryRow(c, "select id,name,email from users where id=$1", claims.UserID).Scan(&u.ID, &u.Name, &u.Email); err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "unknown user"})
			return
		}
		c.Set("user", u)
		c.Next()
	}
}
func (s *Server) issueTokens(ctx context.Context, uid string) (gin.H, error) {
	access := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{uid, jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now())}})
	at, _ := access.SignedString([]byte(s.cfg.JWTSecret))
	rt := randString(48)
	_, err := s.db.Exec(ctx, "insert into sessions(user_id,refresh_token_hash,expires_at) values($1,$2,now()+interval '30 days')", uid, hashToken(rt))
	return gin.H{"accessToken": at, "refreshToken": rt}, err
}
func (s *Server) can(c *gin.Context, wid, need string) bool {
	var role string
	err := s.db.QueryRow(c, "select role from workspace_members where workspace_id=$1 and user_id=$2", wid, userID(c)).Scan(&role)
	if err != nil || rank(role) < rank(need) {
		c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
		return false
	}
	return true
}
func (s *Server) canProject(c *gin.Context, pid, need string) bool {
	var wid string
	if err := s.db.QueryRow(c, "select workspace_id from projects where id=$1", pid).Scan(&wid); err != nil {
		c.AbortWithStatusJSON(404, gin.H{"error": "not found"})
		return false
	}
	return s.can(c, wid, need)
}
func (s *Server) canTask(c *gin.Context, tid, need string) bool {
	var wid string
	err := s.db.QueryRow(c, "select p.workspace_id from tasks t join projects p on p.id=t.project_id where t.id=$1", tid).Scan(&wid)
	if err != nil {
		c.AbortWithStatusJSON(404, gin.H{"error": "not found"})
		return false
	}
	return s.can(c, wid, need)
}
func userID(c *gin.Context) string { u := mustUser(c); return u.ID }
func mustUser(c *gin.Context) User { return c.MustGet("user").(User) }
func (s *Server) projectIDForTask(ctx context.Context, tid string) string {
	var pid string
	_ = s.db.QueryRow(ctx, "select project_id from tasks where id=$1", tid).Scan(&pid)
	return pid
}
func (s *Server) projectEvent(ctx context.Context, pid, typ string, data any) {
	var wid string
	_ = s.db.QueryRow(ctx, "select workspace_id from projects where id=$1", pid).Scan(&wid)
	s.activity(ctx, wid, &pid, typ, data)
	b, _ := json.Marshal(Event{Type: typ, ProjectID: pid, ActorID: actor(ctx), Data: data})
	s.redis.Publish(ctx, "project:"+pid, string(b))
}
func (s *Server) activity(ctx context.Context, wid string, pid *string, typ string, data any) {
	b, _ := json.Marshal(data)
	_, _ = s.db.Exec(ctx, "insert into activity_events(workspace_id,project_id,actor_id,event_type,metadata) values($1,$2,nullif($3,'')::uuid,$4,$5)", wid, pid, actor(ctx), typ, b)
}
func actor(ctx context.Context) string {
	if gc, ok := ctx.(*gin.Context); ok {
		return userID(gc)
	}
	return ""
}
func (s *Server) queueEmail(ctx context.Context, to, subj, body string) {
	_, _ = s.db.Exec(ctx, "insert into email_jobs(to_email,subject,body) values($1,$2,$3)", to, subj, body)
}
func (s *Server) notify(ctx context.Context, userID, typ, title, body string) {
	s.notifyRef(ctx, userID, typ, title, body, "", "")
}
func (s *Server) notifyRef(ctx context.Context, userID, typ, title, body, resourceType, resourceID string) {
	_, _ = s.db.Exec(ctx, "insert into notifications(user_id,type,title,body,resource_type,resource_id) values($1,$2,$3,$4,nullif($5,''),nullif($6,'')::uuid)", userID, typ, title, body, resourceType, resourceID)
	var email string
	if err := s.db.QueryRow(ctx, "select email from users where id=$1", userID).Scan(&email); err == nil {
		s.queueEmail(ctx, email, title, body)
	}
}
func (s *Server) ProcessEmailJobs(ctx context.Context) error {
	rows, err := s.db.Query(ctx, "select id,to_email,subject,body from email_jobs where sent_at is null and attempts<5 order by created_at limit 10")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, to, subj, body string
		rows.Scan(&id, &to, &subj, &body)
		log.Printf("email to=%s subject=%s body=%s", to, subj, body)
		s.db.Exec(ctx, "update email_jobs set sent_at=now(), attempts=attempts+1 where id=$1", id)
	}
	return nil
}
func (s *Server) putObject(ctx context.Context, fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	key := "attachments/" + uuid.NewString() + "/" + fh.Filename
	_, err = s.s3.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.cfg.S3Bucket), Key: aws.String(key), Body: f, ContentType: aws.String(fh.Header.Get("Content-Type"))})
	return key, err
}

func bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return true
	}
	return false
}
func fail(c *gin.Context, code int, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		code = 404
	}
	// Avoid leaking raw database errors to clients in production-style responses.
	// Map common errors to user-safe messages while keeping 400/409 for validation.
	msg := err.Error()
	switch {
	case strings.Contains(msg, "duplicate key"):
		msg = "resource already exists"
	case strings.Contains(msg, "foreign key"):
		msg = "referenced resource not found"
	case strings.Contains(msg, "violates check"):
		msg = "invalid value for resource field"
	case code >= 500:
		msg = "internal server error"
	}
	c.JSON(code, gin.H{"error": msg})
}
func scanRows(c *gin.Context, rows pgx.Rows, err error) {
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer rows.Close()
	f := rows.FieldDescriptions()
	out := []map[string]any{}
	var scanErr error
	for rows.Next() {
		vals := make([]any, len(f))
		ptr := make([]any, len(f))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		if err := rows.Scan(ptr...); err != nil {
			scanErr = err
			// Drain remaining rows so connection reuse is not disrupted,
			// but stop appending to the result set.
			continue
		}
		m := map[string]any{}
		for i, fd := range f {
			k := string(fd.Name)
			switch v := vals[i].(type) {
			case nil, string, int16, int32, int64, float32, float64, bool, []string:
				m[k] = v
			case []byte:
				m[k] = string(v)
			case [16]byte:
				m[k] = uuidFromBytes(v)
			case time.Time:
				m[k] = v.Format(time.RFC3339)
			default:
				m[k] = fmt.Sprint(v)
			}
		}
		out = append(out, m)
	}
	if scanErr != nil {
		fail(c, 500, scanErr)
		return
	}
	if err := rows.Err(); err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(200, out)
}
func uuidFromBytes(b [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func atoi(s string) int { i, _ := strconv.Atoi(s); return i }
