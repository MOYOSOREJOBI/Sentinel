import { useTranslation } from "react-i18next";
import { Search, Filter, CalendarClock, Globe, AlertTriangle, ShieldCheck } from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";

export default function TopBar() {
  const { t } = useTranslation();

  return (
    <div className="h-14 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 z-10 sticky top-0 flex items-center px-4 gap-4">
      <div className="relative flex-1 max-w-md hidden md:flex">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          type="search"
          placeholder={t("Search...")}
          className="w-full pl-9 bg-card border-border/50 h-9"
        />
      </div>

      <div className="flex items-center gap-2 overflow-x-auto no-scrollbar ml-auto">
        <Select defaultValue="1h">
          <SelectTrigger className="w-[110px] h-8 bg-card border-border/50 text-xs font-medium">
            <CalendarClock className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
            <SelectValue placeholder="Time" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="now">{t("Now")}</SelectItem>
            <SelectItem value="1h">{t("1h")}</SelectItem>
            <SelectItem value="24h">{t("24h")}</SelectItem>
            <SelectItem value="7d">{t("7d")}</SelectItem>
            <SelectItem value="custom">{t("Custom")}</SelectItem>
          </SelectContent>
        </Select>

        <Select defaultValue="all">
          <SelectTrigger className="w-[120px] h-8 bg-card border-border/50 text-xs font-medium hidden sm:flex">
            <Globe className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
            <SelectValue placeholder="Asset Class" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Assets</SelectItem>
            <SelectItem value="equities">Equities</SelectItem>
            <SelectItem value="crypto">Crypto</SelectItem>
            <SelectItem value="fx">FX</SelectItem>
          </SelectContent>
        </Select>

        <Select defaultValue="all">
          <SelectTrigger className="w-[110px] h-8 bg-card border-border/50 text-xs font-medium hidden lg:flex">
            <AlertTriangle className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
            <SelectValue placeholder="Severity" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Severities</SelectItem>
            <SelectItem value="critical">Critical</SelectItem>
            <SelectItem value="high">High Risk</SelectItem>
            <SelectItem value="elevated">Elevated</SelectItem>
          </SelectContent>
        </Select>

        <Select defaultValue="all">
          <SelectTrigger className="w-[110px] h-8 bg-card border-border/50 text-xs font-medium hidden xl:flex">
            <ShieldCheck className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
            <SelectValue placeholder="Trust State" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All States</SelectItem>
            <SelectItem value="reliable">Stable</SelectItem>
            <SelectItem value="unreliable">Unreliable</SelectItem>
          </SelectContent>
        </Select>

        <Button variant="outline" size="sm" className="h-8 border-border/50 bg-card hidden sm:flex">
          <Filter className="w-3.5 h-3.5 mr-2" />
          Advanced
        </Button>
      </div>
    </div>
  );
}