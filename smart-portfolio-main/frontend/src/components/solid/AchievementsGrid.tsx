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
          <article class="border rounded-[50px] p-4 min-w-100 min-h-140 flex justify-between items-start gap-4 bg-[#fc785d]">
            <div class="max-h-50 flex flex-col py-7 px-6 gap-2 max-w-90">
              <div class="flex justify-between items-center">
                <span class="bg-[#e1e1e1d0] w-10 min-h-[1.5px] mb-8 rounded-2xl translate-y-4"></span>
                <span class="font-semibold text-sm text-[#e1e1e1] border-[1.5px] border-[#e1e1e1d0] rounded-4xl px-2.5 py-1">2020</span>
              </div>
              <span class="text-4xl font-bold text-[#e1e1e1] mt-6">{item.metric}</span>
              <h4 class="font-semibold text-[#d6d6d6]">{item.title}</h4>
              <p class="text-gray-800 text-sm mt-4 min-h-15">{item.description}</p>
              <div class="rounded-[40px] bg-amber-200 min-w-80 min-h-60">
                <img src="" alt="" />
              </div>
            </div>
            
          </article>
        )}
      </For>
    </div>
  );
}
