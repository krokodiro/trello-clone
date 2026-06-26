export type User = {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  email_verified_at?: string;
  is_admin?: boolean;
  created_at: string;
};

export type WorkspaceRole = "owner" | "admin" | "member";

export type Workspace = {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
};

export type WorkspaceMember = {
  workspace_id: string;
  user_id: string;
  role: WorkspaceRole;
  joined_at: string;
  user?: User;
};

export type Board = {
  id: string;
  workspace_id: string;
  name: string;
  description?: string;
  position: number;
  created_at: string;
  updated_at: string;
};

export type Comment = {
  id: string;
  task_id: string;
  user_id: string;
  body: string;
  created_at: string;
  updated_at: string;
  user?: User;
};

export type Task = {
  id: string;
  list_id: string;
  title: string;
  description?: string;
  position: number;
  due_date?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  assignees?: User[];
  comments?: Comment[];
};

export type List = {
  id: string;
  board_id: string;
  name: string;
  position: number;
  created_at: string;
  updated_at: string;
  tasks?: Task[];
};

export type BoardDetail = {
  board: Board;
  lists: List[];
};

export type Notification = {
  id: string;
  user_id: string;
  type: string;
  title: string;
  body: string;
  link: string;
  read: boolean;
  created_at: string;
};

export type TokenPair = {
  access_token: string;
  refresh_token: string;
};

export type WSEvent = {
  type: string;
  board_id: string;
  client_id?: string;
  payload: unknown;
};
