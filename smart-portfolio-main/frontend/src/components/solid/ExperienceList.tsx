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
    <div id="experience-list" class="app-scrollbar mt-6 overflow-x-hidden">
      <For each={items()}>
        {(item) => (
          <article class="article border-t gap-1 mb-8 flex flex-col justify-center" style="border-top: 2px solid; border-image: linear-gradient(to right, transparent, #EB9A6C, transparent) 1;">
            <div class="flex justify-between mx-4 md:mx-20 lg:mx-42 mt-2 gap-4">
              <div class="">
                <h4 class="text-base md:text-lg font-semibold font-sans leading-tight">{item.company}</h4>
              </div>
              <span class="text-xs md:text-sm font-extralight whitespace-nowrap">{item.period}</span>
            </div>

            <div class="mx-4 md:mx-20 lg:mx-42 items-end flex flex-col">
              <p class="font-medium text-end text-[#ff4d00] text-sm md:text-base">{item.role}</p>
              <p class="text-gray-600 pl-10 md:pl-20 max-w-full lg:max-w-130 text-end text-xs md:text-sm">{item.summary}</p>
            </div>

            <div class="mx-4 md:mx-20 lg:mx-42 mt-2 flex gap-2 flex-wrap justify-end">
              <For each={item.stack ?? []}>
                {(tech) => (
                  <span class="bg-[#fe5f1be2] text-white text-[10px] md:text-xs font-semibold px-2 py-1 rounded">
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