export interface User {
  id: string;
  name: string;
  email: string;
}
export interface Note {
  id: string;
  title: string;
  body?: string;
  tags: string[];
  pinned: boolean;
  created_at: string;
  updated_at: string;
}
export interface SettingsData {
  theme: "system" | "light" | "dark";
  page_size: number;
}
export interface SharedProps {
  [key: string]: unknown;
  auth: User | null;
  theme?: SettingsData["theme"];
  appName?: string;
  errors: Record<string, string>;
}
export const date = (value: string) =>
  new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
