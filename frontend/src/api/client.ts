import axios from 'axios';
import { useAuthStore } from '../lib/store';
export const API_URL = import.meta.env.VITE_API_URL || '/api';
export const api = axios.create({ baseURL: API_URL });
api.interceptors.request.use((config)=>{ const token=useAuthStore.getState().accessToken; if(token) config.headers.Authorization=`Bearer ${token}`; return config; });
api.interceptors.response.use(r=>r, async err=>{ const original=err.config; const store=useAuthStore.getState(); if(err.response?.status===401 && !original._retry && store.refreshToken){ original._retry=true; const {data}=await axios.post(`${API_URL}/auth/refresh`,{refreshToken:store.refreshToken}); store.setTokens(data.tokens); original.headers.Authorization=`Bearer ${data.tokens.accessToken}`; return api(original);} throw err; });
export type Workspace={id:string;name:string;slug:string;role:string};export type Project={id:string;name:string;description:string;icon?:string};export type Task={id:string;title:string;description?:string;priority:string;assigneeId?:string;assigneeIds?:string[];dueDate?:string;labels?:string[];position:number};export type Column={id:string;name:string;position:number;tasks:Task[]};export type AISignal={label:string;value:string;severity:'low'|'medium'|'high'};export type AIAnalyzeResponse={answer:string;signals:AISignal[];suggestedActions:string[]};
