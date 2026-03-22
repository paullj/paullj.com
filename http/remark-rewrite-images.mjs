import { visit } from "unist-util-visit";

export function remarkRewriteImages() {
  return (tree) => {
    visit(tree, "image", (node) => {
      if (node.url && node.url.startsWith("content/images/")) {
        node.url = "/" + node.url;
      }
    });
  };
}
