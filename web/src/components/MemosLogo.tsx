import { useInstance } from "@/contexts/InstanceContext";
import { cn } from "@/lib/utils";
import UserAvatar from "./UserAvatar";

interface Props {
  className?: string;
  collapsed?: boolean;
}

function MemosLogo(props: Props) {
  const { collapsed } = props;
  const { generalSetting: instanceGeneralSetting } = useInstance();
  const title = instanceGeneralSetting.customProfile?.title || "Memos";
  const avatarUrl = instanceGeneralSetting.customProfile?.logoUrl || "/logo.webp";

  return (
    <div className={cn("relative w-full h-auto shrink-0", props.className)}>
      <div className={cn("w-auto flex flex-row justify-start items-center text-foreground", collapsed ? "px-1" : "px-2")}>
        {collapsed ? (
          <UserAvatar className="shrink-0" avatarUrl={avatarUrl} />
        ) : (
          <>
            <img src="/logo.webp" alt={title} className="h-10 w-auto object-contain" />
            <span className="text-xl font-bold text-foreground tracking-tight ml-2">{title}</span>
          </>
        )}
      </div>
    </div>
  );
}

export default MemosLogo;
