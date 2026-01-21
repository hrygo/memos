import { GlobeIcon } from "lucide-react";
import { FC } from "react";
import { Button } from "@/components/ui/button";
import { loadLocale } from "@/utils/i18n";

interface Props {
  value: Locale;
  onChange: (locale: Locale) => void;
}

const LocaleSelect: FC<Props> = (props: Props) => {
  const { onChange, value } = props;

  const handleLocaleChange = () => {
    const nextLocale = value === "en" ? "zh-Hans" : "en";
    loadLocale(nextLocale);
    onChange(nextLocale);
  };

  return (
    <Button variant="outline" className="w-auto" onClick={handleLocaleChange}>
      <GlobeIcon className="w-4 h-auto mr-2" />
      {value === "en" ? "中文(简体)" : "English"}
    </Button>
  );
};

export default LocaleSelect;
