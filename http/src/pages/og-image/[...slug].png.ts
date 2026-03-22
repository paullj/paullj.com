import type { APIContext, GetStaticPaths } from "astro";
import { getCollection } from "astro:content";
import satori, { type SatoriOptions } from "satori";
import { html } from "satori-html";
import { Resvg } from "@resvg/resvg-js";
import { siteConfig } from "../../site.config";
import fs from "node:fs";
import path from "node:path";

const JetBrainsMono = fs.readFileSync(
  path.resolve(process.cwd(), "src/assets/JetBrainsMono-Regular.ttf"),
);
const JetBrainsMonoBold = fs.readFileSync(
  path.resolve(process.cwd(), "src/assets/JetBrainsMono-Bold.ttf"),
);

const ogOptions: SatoriOptions = {
  width: 1200,
  height: 630,
  fonts: [
    { name: "JetBrains Mono", data: JetBrainsMono, weight: 400, style: "normal" },
    { name: "JetBrains Mono", data: JetBrainsMonoBold, weight: 700, style: "normal" },
  ],
};

function formatDate(date: Date) {
  return date.toLocaleDateString("en-GB", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

const markup = (title: string, pubDate: string) =>
  html`<div tw="flex flex-col w-full h-full bg-[#1d1f21] text-[#c9cacc]">
    <div tw="flex flex-col flex-1 w-full p-10 justify-center">
      <p tw="text-2xl mb-6">${pubDate}</p>
      <h1 tw="text-6xl font-bold leading-snug text-white">${title}</h1>
    </div>
    <div tw="flex items-center justify-between w-full p-10 border-t border-[#2bbc89] text-xl">
      <div tw="flex items-center">
        <p tw="ml-3 font-semibold">${siteConfig.title}</p>
      </div>
      <p>by ${siteConfig.author}</p>
    </div>
  </div>`;

export async function GET({ params: { slug } }: APIContext) {
  const posts = await getCollection("posts");
  const post = posts.find((p) => p.id === slug);
  const title = post?.data.title ?? siteConfig.title;
  const postDate = formatDate(post?.data.updatedAt ?? post?.data.date ?? new Date());
  const svg = await satori(markup(title, postDate), ogOptions);
  const png = new Resvg(svg).render().asPng();
  return new Response(png, {
    headers: {
      "Content-Type": "image/png",
      "Cache-Control": "public, max-age=31536000, immutable",
    },
  });
}

export const getStaticPaths: GetStaticPaths = async () => {
  const posts = await getCollection("posts");
  return posts
    .filter(({ data }) => !data.ogImage)
    .map(({ id }) => ({ params: { slug: id } }));
};
