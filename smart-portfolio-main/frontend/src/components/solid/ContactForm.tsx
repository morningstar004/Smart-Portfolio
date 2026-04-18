/** @jsxImportSource solid-js */
import { createSignal } from "solid-js";
import { apiUrl } from "../../lib/public-api";

export default function ContactForm() {
  const [statusText, setStatusText] = createSignal("");
  const [statusClass, setStatusClass] = createSignal(
    "text-[10px] font-mono text-muted-foreground uppercase tracking-widest min-h-[1.5rem]",
  );
  const [submitting, setSubmitting] = createSignal(false);
  const [buttonActive, setButtonActive] = createSignal(false);

  const hasValidInput = (form: HTMLFormElement): boolean => {
    const inputs = Array.from(
      form.querySelectorAll("input[required], textarea[required]"),
    ) as Array<HTMLInputElement | HTMLTextAreaElement>;

    return inputs.every((input) => input.value.trim() !== "");
  };

  const handleMouseEnter = (event: MouseEvent): void => {
    const button = event.currentTarget as HTMLButtonElement | null;
    const form = button?.form;
    if (!button || !form || submitting() || !hasValidInput(form)) return;
    setButtonActive(true);
  };

  const handleMouseLeave = (): void => {
    setButtonActive(false);
  };

  const handleSubmit = async (event: SubmitEvent): Promise<void> => {
    event.preventDefault();
    const form = event.currentTarget as HTMLFormElement | null;
    if (!form) return;

    setSubmitting(true);
    setButtonActive(false);
    setStatusText(">> UPLOADING_PAYLOAD...");
    setStatusClass(
      "text-[10px] font-mono text-brand-orange uppercase tracking-widest min-h-[1.5rem]",
    );

    const formData = new FormData(form);
    const payload = {
      sender_name: String(formData.get("sender_name") ?? ""),
      sender_email: String(formData.get("sender_email") ?? ""),
      message_body: String(formData.get("message_body") ?? ""),
    };

    try {
      const res = await fetch(apiUrl("/api/contact"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const envelope = await res.json();

      if (envelope.success) {
        setStatusText(">> SUCCESS: MESSAGE_STATION_ACKNOWLEDGED_200_OK");
        setStatusClass(
          "text-[10px] font-mono text-brand-green uppercase tracking-widest font-bold min-h-[1.5rem]",
        );
        form.reset();
      } else {
        setStatusText(
          `>> ERROR: ${envelope.error?.message ?? "UNKNOWN_EXCEPTION"}`,
        );
        setStatusClass(
          "text-[10px] font-mono text-brand-red uppercase tracking-widest min-h-[1.5rem]",
        );
      }
    } catch {
      setStatusText(">> FATAL: NETWORK_HANDSHAKE_FAILED");
      setStatusClass(
        "text-[10px] font-mono text-brand-red uppercase tracking-widest min-h-[1.5rem]",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      id="contact-form"
      class="space-y-6 md:space-y-8 w-full"
      onSubmit={(event) => void handleSubmit(event)}
    >
      <div>
        <h2 class="text-3xl text-center font-bold text-gray-900 max-md:text-2xl max-sm:text-xl dark:text-gray-300">
          Let's Talk Tech
        </h2>
        <p class="text-center text-gray-600 max-md:text-sm dark:text-gray-400">
          Fill out the form below and I'll get back to you as soon as possible!
        </p>
      </div>
      <div class="flex flex-col justify-around md:grid md:grid-cols-2 gap-4 md:gap-8">
        <div class="space-y-2 group">
          <input
            type="text"
            id="sender_name"
            name="sender_name"
            required
            placeholder="Your Name"
            class="w-6/7 bg-transparent border-b translate-x-10 border-border rounded-[16px] py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground transition-colors max-md:w-full max-md:translate-x-0"
          />
        </div>
        <div class="space-y-2 group">
          <input
            type="email"
            id="sender_email"
            name="sender_email"
            required
            placeholder="Your Email"
            class="w-6/7 bg-transparent border-b border-border rounded-[16px] py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground transition-colors max-md:w-full"
          />
        </div>
      </div>
      <div class="space-y-2 group items-center flex">
        <textarea
          id="message_body"
          name="message_body"
          required
          placeholder="Your Message"
          rows="4"
          class="w-7/8 h-30 bg-transparent border-b border-border py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground resize-none transition-colors translate-x-10 rounded-[25px] max-md:w-full max-md:translate-x-0"
        ></textarea>
      </div>

      <div id="form-status" class={statusClass()}>
        {statusText()}
      </div>
      <div class="flex justify-center pt-2">
        <button
          class="group relative cursor-pointer overflow-hidden rounded-full border bg-[#fffcf3] p-2 px-6 text-center font-semibold text-black dark:bg-[#0a0a0a] dark:text-white dark:border-[#fffcf3] transition-all duration-300 min-w-[120px] h-[44px] flex items-center justify-center w-full sm:w-auto"
          type="submit"
          id="contact-submit"
          disabled={submitting()}
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
        >
          <div
            class="flex items-center justify-center w-full h-full"
            style="opacity: 1;"
          >
            <div class="absolute left-1/2 top-1/2 h-2 w-2 -translate-x-1/2 -translate-y-1/2 rounded-full bg-black transition-all duration-300 delay-200 opacity-0 group-hover:scale-[100] group-hover:opacity-100 dark:bg-[#fffcf3] z-0"></div>
            <div class="relative z-10 flex items-center gap-3 transition-all duration-300 delay-200 group-hover:-translate-x-8 group-hover:opacity-0 w-full justify-center">
              <div class={`h-2 w-2 rounded-full bg-black dark:bg-[#fffcf3] transition-all duration-300 ${
          buttonActive()
            ? "bg-green-400 text-white"
            : "bg-foreground text-background"
        } `}></div>
              <span class="inline-block relative z-10 text-sm sm:text-base">{submitting() ? "TRANSMITTING..." : "Submit Inquiry"}{" "}</span>
            </div>
            <div class="absolute inset-0 z-20 flex items-center justify-center gap-2 text-white translate-x-8 opacity-0 transition-all duration-300 delay-200 group-hover:translate-x-0 group-hover:opacity-100 dark:text-black">
              <span class="flex items-center justify-center gap-2 w-full text-sm sm:text-base">
                {submitting() ? "TRANSMITTING..." : "Submit Inquiry"}{" "}
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
                  class="lucide lucide-arrow-right h-4 w-4"
                  aria-hidden="true"
                >
                  <path d="M5 12h14"></path>
                  <path d="m12 5 7 7-7 7"></path>
                </svg>
              </span>
            </div>
          </div>
        </button>
      </div>
    </form>
  );
}