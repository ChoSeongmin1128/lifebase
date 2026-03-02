import { cn } from "@/lib/utils";

type PageToolbarProps = React.HTMLAttributes<HTMLDivElement>;

export function PageToolbar({ className, ...props }: PageToolbarProps) {
  return (
    <div
      className={cn(
        "flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 md:px-6 py-3",
        className
      )}
      {...props}
    />
  );
}

type PageToolbarGroupProps = React.HTMLAttributes<HTMLDivElement>;

export function PageToolbarGroup({ className, ...props }: PageToolbarGroupProps) {
  return <div className={cn("flex items-center gap-2 md:gap-3", className)} {...props} />;
}
