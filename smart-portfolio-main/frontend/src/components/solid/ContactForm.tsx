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
        setStatusText(`>> ERROR: ${envelope.error?.message ?? "UNKNOWN_EXCEPTION"}`);
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
    <form id="contact-form" class="space-y-6 md:space-y-8 w-full" onSubmit={(event) => void handleSubmit(event)}>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6 md:gap-8">
        <div class="space-y-2 group">
          <label for="sender_name" class="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">{">> "}SENDER_NAME</label>
          <input
            type="text"
            id="sender_name"
            name="sender_name"
            required
            placeholder="INPUT_NAME"
            class="w-full bg-transparent border-b border-border py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground transition-colors"
          />
        </div>
        <div class="space-y-2 group">
          <label for="sender_email" class="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">{">> "}SENDER_EMAIL</label>
          <input
            type="email"
            id="sender_email"
            name="sender_email"
            required
            placeholder="INPUT_EMAIL"
            class="w-full bg-transparent border-b border-border py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground transition-colors"
          />
        </div>
      </div>
      <div class="space-y-2 group">
        <label for="message_body" class="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">{">> "}MESSAGE_PAYLOAD</label>
        <textarea
          id="message_body"
          name="message_body"
          required
          placeholder="ENTER_MESSAGE_BODY..."
          rows="4"
          class="w-full bg-transparent border-b border-border py-2 text-sm font-mono text-foreground placeholder:text-muted-foreground resize-none transition-colors"
        ></textarea>
      </div>

      <div id="form-status" class={statusClass()}>
        {statusText()}
      </div>

      <button
        type="submit"
        id="contact-submit"
        disabled={submitting()}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        class={`px-8 py-3 font-mono text-xs font-bold tracking-[0.2em] transition-all uppercase border border-border ${
          buttonActive()
            ? "bg-brand-green text-white"
            : "bg-foreground text-background"
        }`}
      >
        {submitting() ? "TRANSMITTING..." : "TRANSMIT_MESSAGE"}
      </button>
    </form>
  );
}
