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
            'From the resume, return ONLY valid JSON as an array of notable achievement objects. Each object must have: title, metric, description. Keep title and metric short. Keep description to one short sentence. Prefer measurable or high-signal outcomes. If exact metrics are unavailable, summarize conservatively. No markdown. No explanation.',
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
          metric: String(item.metric || "KEY_SIGNAL").toUpperCase().replace(/\s+/g, "_"),
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
    <div id="achievement-grid" class="flex flex-row mt-14 justify-center gap-6 w-full max-w-3xl mx-auto">
      <For each={items()}>
        {(item) => (
          <article class="border rounded-[50px] p-4 min-w-100 min-h-140 flex justify-between items-start gap-4">
            <div class="max-h-50 flex flex-col py-10 px-4 gap-2 max-w-90">
              <div class="flex justify-between items-center">
                <span class="bg-[#404040d0] w-10 min-h-[1.5px] mb-8 rounded-2xl"></span>
                <span class="font-semibold text-sm border-[1.5px] border-[#404040d0] rounded-4xl px-2.5 py-1">2020</span>
              </div>
              <span class="text-3xl font-bold text-[#ec7a3d]">{item.metric}</span>
              <h4 class="text-lg font-semibold text-gray-900">{item.title}</h4>
              <p class="text-gray-700 text-sm mt-4">{item.description}</p>
            </div>
            
          </article>
        )}
      </For>
    </div>
  );
}
