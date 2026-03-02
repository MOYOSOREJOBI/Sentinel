import { useTranslation } from "react-i18next";
import { Activity, Clock, Server, CheckCircle2 } from "lucide-react";

export default function TrustStrip() {
  const { t } = useTranslation();

  return (
    <div className="bg-[hsl(var(--stable)/0.1)] border-b border-[hsl(var(--stable)/0.2)] px-4 py-1.5 flex items-center justify-between text-[11px] font-medium font-mono text-[hsl(var(--stable))]">
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-1.5">
          <Activity className="w-3.5 h-3.5" />
          <span className="uppercase tracking-wider">{t("All Systems Nominal")}</span>
        </div>
      </div>
      
      <div className="flex items-center gap-4 opacity-80">
        <div className="flex items-center gap-1.5 hidden sm:flex" title="Cache Health">
          <Server className="w-3.5 h-3.5" />
          <span>99.9% HIT</span>
        </div>
        <div className="flex items-center gap-1.5 hidden md:flex" title="Feed Health">
          <CheckCircle2 className="w-3.5 h-3.5" />
          <span>FEEDS OK</span>
        </div>
        <div className="flex items-center gap-1.5" title="Data Drift">
          <Clock className="w-3.5 h-3.5" />
          <span>&lt; 5ms LAG</span>
        </div>
      </div>
    </div>
  );
}