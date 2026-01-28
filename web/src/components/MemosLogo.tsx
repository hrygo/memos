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
            <img src="/full-logo-light.svg" alt={title} className="h-10 w-auto object-contain dark:hidden" />
            <img src="/full-logo-dark.svg" alt={title} className="h-10 w-auto object-contain hidden dark:block" />
          </>
        )}
      </div>
    </div>
  );
}

export default MemosLogo;
