import { Slot } from "@radix-ui/react-slot";
import type { ButtonHTMLAttributes } from "react";
import {
  buttonVariants,
  type ButtonSize,
  type ButtonVariant,
} from "@/components/ui/button-variants";
import { cn } from "@/lib/cn";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  asChild?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
}

export function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : "button";
  return (
    <Comp
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
}
