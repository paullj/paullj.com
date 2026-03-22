import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { parse } from "yaml";

const raw = readFileSync(resolve(process.cwd(), "../config.yaml"), "utf-8");
const config = parse(raw);

export const siteConfig = {
  author: config.content.name as string,
  title: config.http.title as string,
  description: config.content.description as string,
  lang: config.http.lang as string,
  ogLocale: config.http.og_locale as string,
};
