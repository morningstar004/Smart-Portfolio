/** @jsxImportSource solid-js */
import { For, createSignal, onMount } from "solid-js";
import { apiUrl } from "../../lib/public-api";

export interface AchievementItem {
  year: string;
  title: string;
  metric: string;
  description: string;
  image?: string;
  githubUrl?: string;
}

interface AchievementsGridProps {
  items: AchievementItem[];
}

export default function AchievementsGrid(props: AchievementsGridProps) {
  // 1. Slice initial items to 3
  const [items, setItems] = createSignal(props.items.slice(0, 3));

  const bgColors = ["#E3E1D5", "rgb(255, 95, 95)", "#7EC4CF"];

  const Ylocate = ["translateY(0px)", "translateY(80px)", "translateY(0px)"];

  onMount(() => {
    void loadAchievements();
  });

  async function loadAchievements(): Promise<void> {
    try {
      const res = await fetch(apiUrl("/api/chat"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          question:
            "From the resume, return ONLY valid JSON as an array of notable achievement objects. Each object must have: title, metric, description. Keep title and metric short. Keep description to one short sentence. Prefer measurable or high-signal outcomes. If exact metrics are unavailable, summarize conservatively. No markdown. No explanation.",
        }),
      });

      const envelope = await res.json();
      const raw = envelope?.data?.answer;
      if (!envelope.success || !raw) return;

      const parsed = JSON.parse(raw) as AchievementItem[];
      const normalized = parsed
        .filter((item) => item.title && item.description)
        // 2. Limit the API results to 3 items
        .slice(0, 3)
        .map((item) => ({
          year: String(item.year ?? new Date().getFullYear()), // Provide year, fallback to current year if missing
          title: String(item.title).toUpperCase().replace(/\s+/g, "_"),
          metric: String(item.metric || "KEY_SIGNAL")
            .toUpperCase()
            .replace(/\s+/g, "_"),
          description: String(item.description).trim(),
        }));

      if (normalized.length > 0) {
        setItems(normalized);
      }
    } catch {
      return;
    }
  }

  return (
    <div
      id="achievement-grid"
      class="flex flex-col lg:flex-row mt-12 lg:mt-24 justify-center items-center lg:items-start gap-8 w-full max-w-7xl mx-auto loading:animate-pulse px-12 lg:px-0 max-lg:mt-20"
    >
      <For each={items()}>
        {(item, index) => (
          // Added max-lg:!transform-none to prevent vertical cards from having staggered gaps on mobile
          <div
            class="relative group w-full lg:w-auto max-lg:transform-none!"
            style={{ transform: Ylocate[index() % Ylocate.length] }}
          >
            {/* Hidden decorations on mobile to prevent horizontal overflow */}
            <div class="absolute translate-x-48.25 -translate-y-14 h-4 w-4 rounded-full bg-gray-300 -z-10 max-lg:"></div>
            <div class="absolute translate-x-50 -translate-y-12 w-0.5 h-14 bg-gray-300 -z-10 group-hover:h-16 transition-all duration-500 "></div>
            <article
              class="rounded-[50px] p-4 min-w-100 max-lg:min-w-0 max-lg:w-full min-h-153 max-lg:min-h-136 flex justify-between items-start gap-4 shadow-lg hover:shadow-2xl hover:-translate-y-3 transition-all duration-300 overflow-hidden relative"
              style={{
                "background-color": bgColors[index() % bgColors.length],
              }}
            >
              <div class="flex flex-col py-7 px-6 max-w-90 w-full">
                <div class="flex justify-between items-center">
                  <span class="bg-[#00000060] w-10 min-h-[1.5px] mb-8 translate-y-4"></span>

                  <div class="flex items-center gap-3">
                    {/* GitHub Link Icon */}
                    {item.githubUrl && (
                      <a
                        href={item.githubUrl}
                        target="_blank"
                        class="hover:scale-110 transition-transform mb-8 translate-y-4 text-black/60 hover:text-black"
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          width="24"
                          height="24"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                        >
                          <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"></path>
                          <path d="M9 18c-4.51 2-4.51-2-7-2"></path>
                        </svg>
                      </a>
                    )}
                    <span class="font-semibold text-sm text-black/60 border-[1.5px] rounded-full border-black/60 py-1 px-2 mb-8 translate-y-4">
                      {item.year}
                    </span>
                  </div>
                </div>

                <span class="text-2xl lg:text-4xl font-bold text-black/80 mt-6 wrap-break-words">
                  {item.title}
                </span>
                <h4 class="font-semibold text-black/60 wrap-break-words">
                  {item.metric}
                </h4>
                <p class="text-gray-700 text-sm mt-4 min-h-15">
                  {item.description}
                </p>

                {/* Updated Visual Box for Image */}
                <div class="rounded-[40px] bg-white/30 mt-4 min-w-[20rem] max-lg:min-w-0 max-lg:w-full min-h-60 flex items-center justify-center self-center overflow-hidden border border-white/20">
                  {item.image ? (
                    <img
                      src={item.image}
                      alt={item.title}
                      class="h-62 object-cover group-hover:scale-105 transition-transform duration-500"
                    />
                  ) : (
                    <span class="text-black/20 italic">No Visual</span>
                  )}
                </div>
              </div>
            </article>
          </div>
        )}
      </For>
    </div>
  );
}
