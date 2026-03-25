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
      whileHover={{ y: -1, scale: 1.01 }}
      whileTap={{ scale: 0.99 }}
      transition={{ duration: 0.16, ease: "easeOut" }}
    >
      {children}
    </motion.a>
  );
}
