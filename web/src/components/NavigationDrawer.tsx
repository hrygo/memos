import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { useInstance } from "@/contexts/InstanceContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import { Routes } from "@/router";
import Navigation from "./Navigation";

const NavigationDrawer = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const currentUser = useCurrentUser();
  const [open, setOpen] = useState(false);
  const { generalSetting } = useInstance();
  const title = generalSetting.customProfile?.title || "Memos";

  useEffect(() => {
    setOpen(false);
  }, [location.key]);

  const handleLogoClick = () => {
    if (currentUser) {
      navigate(Routes.CHAT);
    } else {
      navigate(Routes.EXPLORE);
    }
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" className="px-0 hover:bg-transparent cursor-pointer" onClick={handleLogoClick}>
          <img src="/full-logo.webp" alt={title} className="h-10 w-auto object-contain" />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-80 max-w-full overflow-auto px-2 bg-background">
        <SheetHeader>
          <SheetTitle />
        </SheetHeader>
        <Navigation className="pb-4" />
      </SheetContent>
    </Sheet>
  );
};

export default NavigationDrawer;
