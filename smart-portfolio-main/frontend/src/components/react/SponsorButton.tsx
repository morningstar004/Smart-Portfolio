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
      existing.addEventListener(
        "error",
        () => reject(new Error("RAZORPAY_SCRIPT_LOAD_FAILED")),
        {
          once: true,
        },
      );
      return;
    }

    const script = document.createElement("script");
    script.src = "https://checkout.razorpay.com/v1/checkout.js";
    script.async = true;
    script.dataset.razorpayCheckout = "true";
    script.addEventListener("load", () => resolve(), { once: true });
    script.addEventListener(
      "error",
      () => reject(new Error("RAZORPAY_SCRIPT_LOAD_FAILED")),
      {
        once: true,
      },
    );
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
      button.innerText = "SPONSOR ME";

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
      whileTap={{ scale: 0.9 }}
      transition={{ duration: 0.2, ease: "easeIn" }}
      className="relative px-7 py-4 rounded-full overflow-hidden 
             text-white font-mono text-[14px] font-bold tracking-[0.01em]
             backdrop-blur-[24px] border-2 border-white/60 border-b-2 border-b-white/20
             border-r-white/20 shadow-lg
             flex items-center gap-2 uppercase transition-all opacity-90 "
      style={{
        background: `radial-gradient(ellipse at center, 
      rgba(8, 8, 8, 1) 0%, 
      rgba(75, 76, 75, 0.9) 45%, 
      rgba(134, 135, 134, 0.8) 70%, 
      rgba(188, 188, 188, 0.7) 100%
    )`,
      }}
      type="button"
    >
      {/* The "Frosted Grain" Texture Overlay */}
      <div className="absolute inset-0 opacity-[0.5] pointer-events-none bg-[url('https://grainy-gradients.vercel.app/noise.svg')] brightness-50 contrast-50"></div>

      {/* Top Specular Highlight (Makes the glass look thick) */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[90%] h-[1px] bg-gradient-to-r from-transparent via-white/50 to-transparent"></div>

      <span className="relative z-10 drop-shadow-md">SPONSOR ME</span>
    </motion.button>
  );
}
