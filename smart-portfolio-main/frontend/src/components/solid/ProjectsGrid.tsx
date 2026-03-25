/** @jsxImportSource solid-js */
import { For, Show, createSignal, onMount } from "solid-js";
import { apiUrl } from "../../lib/public-api";
import type { Project } from "../../lib/types";

export default function ProjectsGrid() {
  const [projects, setProjects] = createSignal<Project[] | null>(null);
  const [error, setError] = createSignal(false);

  onMount(() => {
    void loadProjects();
  });

  async function loadProjects(): Promise<void> {
    try {
      const res = await fetch(apiUrl("/api/projects"));
      const envelope = await res.json();

      if (!envelope.success || !Array.isArray(envelope.data)) {
        setProjects([]);
        return;
      }

      setProjects(envelope.data);
    } catch {
      setError(true);
    }
  }

  return (
    <div id="projects-grid" class="app-scrollbar flex overflow-x-auto snap-x snap-mandatory gap-px bg-border border border-border">
      <Show when={error()}>
        <div class="w-full bg-background p-12 text-center text-brand-orange font-mono text-xs uppercase tracking-widest">
          CRITICAL_ERROR: REMOTE_REGISTRY_OFFLINE
        </div>
      </Show>

      <Show when={!error() && projects() === null}>
        <For each={[1, 2, 3, 4, 5, 6]}>
          {() => (
            <div class="min-w-[18rem] md:min-w-[22rem] h-[20rem] md:h-[22rem] snap-start shrink-0 bg-background p-6 animate-pulse flex flex-col justify-between">
              <div class="space-y-4">
                <div class="h-4 bg-muted w-2/3" />
                <div class="h-12 bg-muted w-full" />
              </div>
              <div class="h-4 bg-muted w-1/2" />
            </div>
          )}
        </For>
      </Show>

      <Show when={!error() && projects() !== null && projects()!.length === 0}>
        <div class="w-full bg-background p-12 text-center text-muted-foreground font-mono text-xs uppercase tracking-widest">
          No matching registry entries found.
        </div>
      </Show>

      <For each={projects() ?? []}>
        {(project) => (
          <article class="min-w-[18rem] md:min-w-[22rem] max-w-[22rem] h-[20rem] md:h-[22rem] snap-start shrink-0 bg-background p-6 hover:bg-muted/50 transition-all flex flex-col justify-between group">
            <div class="space-y-4">
              <div class="flex justify-between items-start">
                <h4 class="font-mono text-lg font-bold tracking-tight text-foreground group-hover:text-brand-green transition-colors">
                  {project.title.toUpperCase()}
                </h4>
                <div class="flex gap-4 text-muted-foreground">
                  <Show when={project.github_url}>
                    <a href={project.github_url} target="_blank" class="hover:text-brand-orange transition-colors">
                      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
                        <path d="M9 18c-4.51 2-5-2-7-2" />
                      </svg>
                    </a>
                  </Show>
                </div>
              </div>
              <p class="text-[11px] leading-relaxed text-muted-foreground font-mono tracking-tight line-clamp-6">
                {project.description}
              </p>
            </div>

            <div class="space-y-4">
              <Show when={project.tech_stack}>
                <div class="flex flex-wrap gap-2">
                  <For each={project.tech_stack?.split(",") ?? []}>
                    {(tech) => (
                      <span class="text-[9px] font-mono text-muted-foreground uppercase border border-border px-1.5 py-0.5 tracking-tighter">
                        {tech.trim()}
                      </span>
                    )}
                  </For>
                </div>
              </Show>
              <div class="pt-4 border-t border-border flex justify-between items-center">
                <span class="text-[9px] font-mono text-zinc-500 uppercase tracking-widest">V.0.1_STABLE</span>
                <Show when={project.live_url}>
                  <a href={project.live_url} target="_blank" class="text-[9px] font-mono text-brand-green hover:underline uppercase tracking-widest">
                    DEPLOYMENT_LINK
                  </a>
                </Show>
              </div>
            </div>
          </article>
        )}
      </For>
    </div>
  );
}
