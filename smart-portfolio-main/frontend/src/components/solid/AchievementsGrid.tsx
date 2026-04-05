/** @jsxImportSource solid-js */
import { For, createSignal, onMount } from "solid-js";
import { apiUrl } from "../../lib/public-api";

export interface AchievementItem {
  title: string;
  metric: string;
  description: string;
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
      class="flex flex-col lg:flex-row mt-12 lg:mt-24 justify-center items-center lg:items-start gap-8 w-full max-w-7xl mx-auto loading:animate-pulse px-12 lg:px-0"
    >
      <For each={items()}>
        {(item, index) => (
          // Added max-lg:!transform-none to prevent vertical cards from having staggered gaps on mobile
          <div class="relative group w-full lg:w-auto max-lg:!transform-none" style={{ "transform": Ylocate[index() % Ylocate.length] }}>
            {/* Hidden decorations on mobile to prevent horizontal overflow */}
            <div class="absolute translate-x-48.25 -translate-y-14 h-4 w-4 rounded-full bg-gray-300 -z-10 max-lg:hidden"></div>
            <div class="absolute translate-x-50 -translate-y-12 w-[2px] h-14 bg-gray-300 -z-10 group-hover:h-16 transition-all duration-500 max-lg:hidden">
            </div>
            <article
              // Added max-lg:min-w-0 and max-lg:min-h-[30rem] for better mobile scaling
              class="rounded-[50px] p-4 min-w-[25rem] max-lg:min-w-0 max-lg:w-full min-h-[36rem] max-lg:min-h-[30rem] flex justify-between items-start gap-4 shadow-lg hover:shadow-2xl hover:-translate-y-3 transition-all duration-300 overflow-hidden"
              style={{ 
                "background-color": bgColors[index() % bgColors.length],
                "transform": "none" 
              }}
            >
              <div class="max-h-[12.5rem] flex flex-col py-7 px-6 gap-2 max-w-[22.5rem] w-full">
                <div class="flex justify-between items-center">
                  <span class="bg-[#00000060] w-10 min-h-[1.5px] mb-8 translate-y-4"></span>
                  <span class="font-semibold text-sm text-black/60 border-[1.5px] rounded-full border-black/60 py-1 px-2">
                    2020
                  </span>
                </div>
                {/* Responsive typography: text-2xl on mobile, text-4xl on desktop */}
                <span class="text-2xl lg:text-4xl font-bold text-black/80 mt-6 break-words">
                  {item.metric}
                </span>
                <h4 class="font-semibold text-black/60 break-words">{item.title}</h4>
                <p class="text-gray-700 text-sm mt-4 min-h-[3.75rem]">
                  {item.description}
                </p>
                {/* Responsive width for the visual box */}
                <div class="rounded-[40px] bg-white/30 mt-4 min-w-[20rem] max-lg:min-w-0 max-lg:w-full min-h-[15rem] flex items-center justify-center">
                   <span class="text-black/20 italic">Visual</span>
                </div>
              </div>
            </article>
          </div>
        )}
      </For>
    </div>
  );
}