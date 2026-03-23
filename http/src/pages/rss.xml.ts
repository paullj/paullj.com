import rss from "@astrojs/rss";
import { getCollection } from "astro:content";
import { content, http } from "../site.config";

export async function GET(context: { site: URL }) {
	const posts = (await getCollection("posts")).sort(
		(a, b) => b.data.date.getTime() - a.data.date.getTime(),
	);

	return rss({
		title: http.title,
		description: content.description,
		site: context.site,
		items: posts.map((post) => ({
			title: post.data.title,
			description: post.data.description,
			pubDate: post.data.date,
			link: `/posts/${post.id}`,
		})),
	});
}
