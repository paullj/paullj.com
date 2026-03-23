// @ts-check
import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import expressiveCode from "astro-expressive-code";
import sitemap from "@astrojs/sitemap";
import rehypeMermaid from "rehype-mermaid";
import rehypeExternalLinks from "rehype-external-links";
import { remarkRewriteImages } from "./remark-rewrite-images.mjs";
import remarkGithubAdmonitions from "remark-github-beta-blockquote-admonitions";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { parse } from "yaml";

const config = parse(
	readFileSync(resolve(process.cwd(), "../config.yaml"), "utf-8"),
);

// https://astro.build/config
export default defineConfig({
	site: config.http_url,
	integrations: [
		expressiveCode({
			themes: ["dracula", "github-light"],
			themeCssSelector: (theme) =>
				theme.name === "dracula"
					? '[data-theme="dark"]'
					: '[data-theme="light"]',
			styleOverrides: {
				codeFontFamily: '"JetBrains Mono", monospace',
				codeFontSize: "0.75rem",
				borderRadius: "4px",
			},
		}),
		sitemap(),
	],
	vite: {
		plugins: [tailwindcss()],
	},
	markdown: {
		remarkPlugins: [
			remarkRewriteImages,
			[
				remarkGithubAdmonitions,
				{
					classNameMaps: {
						block: (title) => [
							"admonition",
							`admonition-${title.toLowerCase()}`,
						],
						title: "admonition-title",
					},
				},
			],
		],
		rehypePlugins: [
			[rehypeMermaid, { strategy: "inline-svg" }],
			[
				rehypeExternalLinks,
				{ target: "_blank", rel: ["nofollow", "noopener", "noreferrer"] },
			],
		],
		remarkRehype: {
			footnoteLabelProperties: { className: [""] },
		},
	},
});
