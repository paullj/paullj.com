import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { parse } from "yaml";

const raw = readFileSync(resolve(process.cwd(), "../config.yaml"), "utf-8");
const config = parse(raw);

export const httpUrl: string = config.http_url;
export const sshAddress: string = config.ssh_address;

export const content: {
  name: string;
  subtitle: string;
  description: string;
  recent_posts_limit: number;
  about_path: string;
  posts_dir: string;
  links: Array<{ name: string; url: string }>;
} = config.content;

export const http: {
  title: string;
  lang: string;
  og_locale: string;
} = config.http;
