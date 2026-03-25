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
    <div id="achievement-grid" class="app-scrollbar grid grid-cols-1 md:grid-cols-2 gap-px bg-border border border-border max-h-[22rem] overflow-y-auto">
      <For each={items()}>
        {(item) => (
          <article class="bg-background p-6 md:p-7 flex flex-col justify-between min-h-56 space-y-8">
            <div class="space-y-4">
              <span class="text-[9px] font-mono text-brand-orange uppercase tracking-[0.2em]">{item.metric}</span>
              <h4 class="text-base font-mono font-bold tracking-tight text-foreground">{item.title}</h4>
              <p class="text-[11px] leading-relaxed text-muted-foreground font-mono tracking-tight">{item.description}</p>
            </div>
            <div class="pt-4 border-t border-border">
              <span class="text-[9px] font-mono text-brand-green uppercase tracking-[0.2em]">STATUS_CONFIRMED</span>
            </div>
          </article>
        )}
      </For>
    </div>
  );
}
