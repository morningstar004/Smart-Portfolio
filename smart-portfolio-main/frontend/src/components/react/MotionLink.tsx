import { motion } from "framer-motion";
import type { PropsWithChildren } from "react";

interface MotionLinkProps extends PropsWithChildren {
  href: string;
  className?: string;
}

export default function MotionLink({
  children,
  href,
  className,
}: MotionLinkProps) {
  return (
    <motion.a
      href={href}
      className={className}
      whileTap={{ scale: 0.96 }}
      transition={{ duration: 0.16, ease: "easeIn" }}
    >
      {children}
    </motion.a>
  );
}
