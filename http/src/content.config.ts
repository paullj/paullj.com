import { defineCollection, z } from "astro:content";
import { glob } from "astro/loaders";

const posts = defineCollection({
  loader: glob({ pattern: "**/*.md", base: "../content/posts" }),
  schema: z.object({
    title: z.string(),
    date: z.coerce.date(),
    description: z.string().optional(),
    draft: z.boolean().optional().default(false),
    updatedAt: z.coerce.date().optional(),
    tags: z.array(z.string()).optional().default([]),
    ogImage: z.string().optional(),
  }),
});

export const collections = { posts };
