/** @jsxImportSource solid-js */
import { For, createSignal, onMount } from "solid-js";
import { apiUrl } from "../../lib/public-api";

export interface ExperienceItem {
  company: string;
  role: string;
  period: string;
  summary: string;
  stack?: string[];
}

interface ExperienceListProps {
  items: ExperienceItem[];
}

export default function ExperienceList(props: ExperienceListProps) {
  const [items, setItems] = createSignal(props.items);

  onMount(() => {
    void loadExperience();
  });

  async function loadExperience(): Promise<void> {
    try {
      const res = await fetch(apiUrl("/api/chat"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          question:
            'From the resume, return ONLY valid JSON as an array of work experience objects. Each object must have: company, role, period, summary, stack. Keep stack as an array of up to 4 short uppercase technologies. Keep summary to one short sentence. If exact data is missing, infer conservatively from the resume. No markdown. No explanation.',
        }),
      });

      const envelope = await res.json();
      const raw = envelope?.data?.answer;
      if (!envelope.success || !raw) return;

      const parsed = JSON.parse(raw) as ExperienceItem[];
      const normalized = parsed
        .filter((item) => item.company && item.role)
        .slice(0, 4)
        .map((item) => ({
          company: String(item.company).toUpperCase().replace(/\s+/g, "_"),
          role: String(item.role).toUpperCase().replace(/\s+/g, "_"),
          period: String(item.period || "TIMELINE_UNSPECIFIED").toUpperCase().replace(/\s+/g, "_"),
          summary: String(item.summary || "").trim(),
          stack: Array.isArray(item.stack)
            ? item.stack.slice(0, 4).map((tech) => String(tech).toUpperCase().replace(/\s+/g, "_"))
            : [],
        }));

      if (normalized.length > 0) {
        setItems(normalized);
      }
    } catch {
      return;
    }
  }

  return (
    <div id="experience-list" class="app-scrollbar grid grid-cols-1 xl:grid-cols-2 gap-px bg-border border border-border max-h-[62svh] overflow-y-auto">
      <For each={items()}>
        {(item) => (
          <article class="bg-card p-6 md:p-8 space-y-5">
            <div class="flex items-start justify-between gap-6">
              <div class="space-y-2">
                <span class="text-[9px] font-mono text-muted-foreground uppercase tracking-[0.2em]">NODE</span>
                <h4 class="text-lg font-mono font-bold tracking-tight text-foreground">{item.company}</h4>
              </div>
              <span class="text-[9px] font-mono text-brand-orange uppercase tracking-[0.2em] whitespace-nowrap">{item.period}</span>
            </div>

            <div class="border-l border-border pl-4 space-y-3">
              <p class="text-[10px] font-mono text-brand-green uppercase tracking-[0.2em]">{item.role}</p>
              <p class="text-[11px] leading-relaxed text-muted-foreground font-mono tracking-tight">{item.summary}</p>
            </div>

            <div class="flex flex-wrap gap-2">
              <For each={item.stack ?? []}>
                {(tech) => (
                  <span class="text-[9px] font-mono text-muted-foreground uppercase border border-border px-2 py-1 tracking-widest">
                    {tech}
                  </span>
                )}
              </For>
            </div>
          </article>
        )}
      </For>
    </div>
  );
}
