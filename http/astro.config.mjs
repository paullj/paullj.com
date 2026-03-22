// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import expressiveCode from 'astro-expressive-code';
import sitemap from '@astrojs/sitemap';
import rehypeMermaid from 'rehype-mermaid';
import rehypeExternalLinks from 'rehype-external-links';
import { remarkRewriteImages } from './remark-rewrite-images.mjs';

// https://astro.build/config
export default defineConfig({
  site: 'https://paullj.com',
  integrations: [
    expressiveCode({
      themes: ['dracula', 'github-light'],
      themeCssSelector: (theme) =>
        theme.name === 'dracula'
          ? '[data-theme="dark"]'
          : '[data-theme="light"]',
      styleOverrides: {
        codeFontFamily: '"JetBrains Mono", monospace',
        codeFontSize: '0.75rem',
        borderRadius: '4px',
      },
    }),
    sitemap(),
  ],
  vite: {
    plugins: [tailwindcss()],
  },
  markdown: {
    remarkPlugins: [remarkRewriteImages],
    rehypePlugins: [
      [rehypeMermaid, { strategy: 'inline-svg' }],
      [rehypeExternalLinks, { target: '_blank', rel: ['nofollow', 'noopener', 'noreferrer'] }],
    ],
    remarkRehype: {
      footnoteLabelProperties: { className: [''] },
    },
  },
});
