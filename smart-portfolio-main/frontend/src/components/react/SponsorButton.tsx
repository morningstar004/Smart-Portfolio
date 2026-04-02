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
      /* Subtle lift without scaling */
      whileHover={{}}
      whileTap={{ scale: 0.96 }}
      transition={{ duration: 0.2, ease: "easeIn" }}
      className="group relative px-8 py-4 rounded-full overflow-hidden
text-white font-mono text-[14px] font-bold tracking-[0.05em]
backdrop-blur-[20px] border-b-[#ffffff]/50 border-t-[#ffffff]
flex items-center justify-center uppercase transition-all duration-300
hover:border-r-[#ffffff]/50 hover:border-l-[#ffffff]/50 hover:border-b-opacity-30 hover:border-t-opacity-30
max-md:px-5 max-md:py-2.5 max-md:text-[12px] max-sm:px-3.5 max-sm:py-2 max-sm:text-[10px] max-sm:hidden max-sm:tracking-normal"
      style={{
        background: `radial-gradient(ellipse at center, 
      rgba(8, 8, 8, 1) 0%, 
      rgba(75, 76, 75, 0.9) 35%, 
      rgba(134, 135, 134, 0.8) 60%, 
      rgba(188, 188, 188, 0.7) 80%,
      rgba(255, 255, 255, 0.6) 100%
    )`,
      }}
      type="button"
    >
      {/* SUBTLE HOVER GLOW: A soft light that follows the mouse (simplified to a brightness shift) */}
      <div className="absolute inset-0 opacity-0 transition-opacity duration-500 bg-white pointer-events-none" />

      {/* GRAIN TEXTURE: Keeps the "thick frost" look */}
      <div className="absolute inset-0 opacity-[0.2] pointer-events-none bg-[url('https://grainy-gradients.vercel.app/noise.svg')] brightness-75 contrast-75"></div>

      {/* TOP RIM LIGHT: Fixed highlight to define the edge */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[85%] h-[1px] bg-gradient-to-r from-transparent via-white/30 to-transparent"></div>

      <span className="relative z-10 drop-shadow-sm max-md:text-[10px] px-1">SPONSOR ME</span>
    </motion.button>
  );
}
