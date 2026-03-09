import { cn } from "@/lib/utils";

type PageToolbarProps = React.HTMLAttributes<HTMLDivElement>;

export function PageToolbar({ className, ...props }: PageToolbarProps) {
  return (
    <div
      className={cn(
        "flex min-h-[64px] flex-wrap items-center justify-between gap-3 border-b border-border bg-background px-4 py-3 md:px-6 lg:px-8",
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
