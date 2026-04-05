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
  const [items, setItems] = createSignal(props.items);

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
        .slice(0, 6)
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
      class="flex flex-row mt-24 justify-center gap-6 w-full max-w-3xl mx-auto loading:animate-pulse"
    >
      <For each={items()}>
        {(item) => (
          <div>
            <div class="absolute -translate-x-56.75 -translate-y-14 h-4 w-4 rounded-full bg-gray-300 -z-10"></div>
            <div class="absolute -translate-x-55 -translate-y-12 w-[2px] h-14 bg-gray-300 -z-10 group-hover:h-16 transition-all duration-500">
            </div>
            <article
              class="rounded-[50px] p-4 min-w-[25rem] min-h-[36rem] flex justify-between items-start gap-4 shadow-lg hover:shadow-2xl hover:-translate-y-3 transition-all duration-300 overflow-hidden"
              style="background-color: rgb(255, 95, 95); transform: none;"
            >
              <div class="max-h-[12.5rem] flex flex-col py-7 px-6 gap-2 max-w-[22.5rem]">
                <div class="flex justify-between items-center">
                  <span class="bg-[#e1e1e1d0] w-10 min-h-[1.5px] mb-8 translate-y-4"></span>
                  <span class="font-semibold text-sm text-[#e1e1e1] border-[1.5px] rounded-full border-[#e1e1e1d0] py-1 px-2">
                    2020
                  </span>
                </div>
                <span class="text-4xl font-bold text-[#e1e1e1] mt-6">
                  {item.metric}
                </span>
                <h4 class="font-semibold text-[#d6d6d6]">{item.title}</h4>
                <p class="text-gray-800 text-sm mt-4 min-h-[3.75rem]">
                  {item.description}
                </p>
                <div class="rounded-[40px] bg-amber-200 mt-4 min-w-[20rem] min-h-[15rem]">
                  <img src="" alt="" />
                </div>
              </div>
            </article>
          </div>
        )}
      </For>
    </div>
  );
}
