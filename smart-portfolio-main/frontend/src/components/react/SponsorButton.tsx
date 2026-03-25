import { motion } from "framer-motion";
import { createRef, useEffect } from "react";
import { apiUrl } from "../../lib/public-api";

declare global {
  interface Window {
    Razorpay?: new (options: Record<string, unknown>) => {
      open: () => void;
    };
  }
}

async function ensureRazorpayLoaded(): Promise<void> {
  if (window.Razorpay) return;

  await new Promise<void>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      'script[data-razorpay-checkout="true"]',
    );
    if (existing) {
      existing.addEventListener("load", () => resolve(), { once: true });
      existing.addEventListener("error", () => reject(new Error("RAZORPAY_SCRIPT_LOAD_FAILED")), {
        once: true,
      });
      return;
    }

    const script = document.createElement("script");
    script.src = "https://checkout.razorpay.com/v1/checkout.js";
    script.async = true;
    script.dataset.razorpayCheckout = "true";
    script.addEventListener("load", () => resolve(), { once: true });
    script.addEventListener("error", () => reject(new Error("RAZORPAY_SCRIPT_LOAD_FAILED")), {
      once: true,
    });
    document.head.appendChild(script);
  });
}

export default function SponsorButton() {
  const buttonRef = createRef<HTMLButtonElement>();

  useEffect(() => {
    const button = buttonRef.current;
    if (!button) return;

    const handleClick = async () => {
      const originalText = button.innerText;
      button.disabled = true;
      button.innerText = "INITIALIZING...";

      try {
        await ensureRazorpayLoaded();

        const response = await fetch(apiUrl("/api/payments/create-order"), {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ amount: 500, currency: "INR" }),
        });

        if (!response.ok) {
          throw new Error("API_ERROR");
        }

        const envelope = await response.json();
        const data = envelope?.data;
        if (!envelope?.success || !data?.id || !data?.key_id) {
          throw new Error("RAZORPAY_CONFIG_MISSING");
        }

        const razorpay = new window.Razorpay!({
          key: data.key_id,
          amount: data.amount,
          currency: data.currency,
          name: "ZR_SPONSOR_HUB",
          description: "SYSTEM_CONTRIBUTION_V1",
          order_id: data.id,
          handler() {
            alert("CONTRIBUTION_RECEIVED_SUCCESSFULLY. THANK_YOU.");
          },
          theme: { color: "#ec4899" },
        });

        razorpay.open();
      } catch (error) {
        console.error(error);
        alert("GATEWAY_FAILED_RETRY_LATER");
      } finally {
        button.disabled = false;
        button.innerText = originalText;
      }
    };

    button.addEventListener("click", handleClick);
    return () => {
      button.removeEventListener("click", handleClick);
    };
  }, []);

  return (
    <motion.button
      ref={buttonRef}
      whileHover={{ y: -1, scale: 1.01 }}
      whileTap={{ scale: 0.99 }}
      transition={{ duration: 0.16, ease: "easeOut" }}
      className="hidden sm:flex items-center gap-2 px-4 py-1.5 border border-border font-mono text-[10px] tracking-widest hover:bg-[#ec4899] hover:text-white transition-all"
      type="button"
    >
      SPONSOR_ME
    </motion.button>
  );
}
