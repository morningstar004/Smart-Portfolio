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
    <div id="experience-list" class="app-scrollbar ">
      <For each={items()}>
        {(item) => (
          <article class="bg-card ">
            <div class="">
              <div class="">
                <span class="">NODE</span>
                <h4 class="">{item.company}</h4>
              </div>
              <span class="">{item.period}</span>
            </div>

            <div class="">
              <p class="">{item.role}</p>
              <p class="">{item.summary}</p>
            </div>

            <div class="">
              <For each={item.stack ?? []}>
                {(tech) => (
                  <span class="">
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
