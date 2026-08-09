import { mkdir, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";

const owner = process.env.GITHUB_REPOSITORY?.split("/")[0] || "heidsoft";
const repo = process.env.GITHUB_REPOSITORY?.split("/")[1] || "itsm";
const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
const output = resolve(process.argv[2] || "docs/assets/star-history.svg");

if (!token) {
  throw new Error("GITHUB_TOKEN or GH_TOKEN is required");
}

const stars = [];
for (let page = 1; ; page += 1) {
  const response = await fetch(
    `https://api.github.com/repos/${owner}/${repo}/stargazers?per_page=100&page=${page}`,
    {
      headers: {
        Accept: "application/vnd.github.star+json",
        Authorization: `Bearer ${token}`,
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": `${repo}-star-history-generator`,
      },
    },
  );

  if (!response.ok) {
    throw new Error(`GitHub API returned ${response.status}: ${await response.text()}`);
  }

  const pageStars = await response.json();
  stars.push(...pageStars);
  if (pageStars.length < 100) break;
}

const dates = stars
  .map((star) => new Date(star.starred_at))
  .filter((date) => !Number.isNaN(date.valueOf()))
  .sort((a, b) => a - b);

if (dates.length === 0) {
  throw new Error("GitHub returned no timestamped stargazers");
}

const width = 900;
const height = 470;
const margin = { top: 50, right: 42, bottom: 65, left: 72 };
const chartWidth = width - margin.left - margin.right;
const chartHeight = height - margin.top - margin.bottom;
const first = dates[0].valueOf();
const last = Math.max(dates.at(-1).valueOf(), first + 86_400_000);
const maxStars = dates.length;
const x = (date) => margin.left + ((date.valueOf() - first) / (last - first)) * chartWidth;
const y = (count) => margin.top + chartHeight - (count / maxStars) * chartHeight;
const points = dates.map((date, index) => `${x(date).toFixed(1)},${y(index + 1).toFixed(1)}`);
points.unshift(`${margin.left},${margin.top + chartHeight}`);

const escapeXml = (value) =>
  value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
const formatDate = (date) => date.toISOString().slice(0, 10);
const yTicks = Array.from(new Set([0, 0.25, 0.5, 0.75, 1].map((n) => Math.round(n * maxStars))));
const xTicks = Array.from({ length: 5 }, (_, index) => new Date(first + ((last - first) * index) / 4));

const grid = yTicks
  .map(
    (tick) => `<g><line x1="${margin.left}" y1="${y(tick)}" x2="${width - margin.right}" y2="${y(tick)}" stroke="#e5e7eb"/><text x="${margin.left - 14}" y="${y(tick) + 5}" text-anchor="end" fill="#64748b">${tick}</text></g>`,
  )
  .join("");
const labels = xTicks
  .map(
    (tick) => `<text x="${x(tick)}" y="${height - 28}" text-anchor="middle" fill="#64748b">${formatDate(tick)}</text>`,
  )
  .join("");

const svg = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-labelledby="title desc">
  <title id="title">${escapeXml(`${owner}/${repo} GitHub Star growth`)}</title>
  <desc id="desc">${maxStars} stars from ${formatDate(dates[0])} to ${formatDate(dates.at(-1))}</desc>
  <rect width="100%" height="100%" rx="12" fill="#ffffff"/>
  <text x="${margin.left}" y="30" font-family="system-ui, sans-serif" font-size="20" font-weight="600" fill="#0f172a">GitHub Star 增长趋势</text>
  <text x="${width - margin.right}" y="30" text-anchor="end" font-family="system-ui, sans-serif" font-size="16" fill="#f59e0b">★ ${maxStars}</text>
  <g font-family="system-ui, sans-serif" font-size="12">${grid}${labels}</g>
  <polygon points="${points.join(" ")} ${width - margin.right},${margin.top + chartHeight}" fill="#fef3c7" opacity="0.7"/>
  <polyline points="${points.join(" ")}" fill="none" stroke="#f59e0b" stroke-width="3" stroke-linejoin="round" stroke-linecap="round"/>
  <circle cx="${x(dates.at(-1))}" cy="${y(maxStars)}" r="5" fill="#f59e0b"/>
  <text x="${width - margin.right}" y="${height - 8}" text-anchor="end" font-family="system-ui, sans-serif" font-size="11" fill="#94a3b8">由 GitHub Actions 自动更新</text>
</svg>
`;

await mkdir(dirname(output), { recursive: true });
await writeFile(output, svg, "utf8");
console.log(`Generated ${output} with ${maxStars} stars.`);
