import { useTranslation } from "react-i18next";
import { Link, useLocation } from "wouter";
import { 
  LayoutDashboard, 
  ListTodo, 
  ShieldAlert, 
  Building2, 
  BarChart3, 
  Info, 
  BookA, 
  Eye,
  Settings,
  ShieldCheck,
  Languages
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

const navItems = [
  { href: "/command-center", icon: LayoutDashboard, label: "Command Center" },
  { href: "/queue", icon: ListTodo, label: "Queue" },
  { href: "/trust", icon: ShieldCheck, label: "Trust" },
  { href: "/governance", icon: Building2, label: "Governance" },
  { href: "/executive", icon: BarChart3, label: "Executive" },
  { href: "/watchlists", icon: Eye, label: "Watchlists" },
];

const bottomItems = [
  { href: "/about", icon: Info, label: "About" },
  { href: "/glossary", icon: BookA, label: "Glossary" },
];

export default function Sidebar() {
  const { t, i18n } = useTranslation();
  const [location] = useLocation();

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    document.documentElement.dir = i18n.dir();
  };

  return (
    <div className="w-16 lg:w-64 border-r border-border bg-card flex flex-col items-center lg:items-start transition-all duration-300 z-20">
      <div className="h-16 w-full flex items-center justify-center lg:justify-start lg:px-6 border-b border-border">
        <ShieldAlert className="w-6 h-6 text-primary lg:mr-2" />
        <span className="hidden lg:block font-semibold text-lg tracking-tight">Sentinel</span>
      </div>

      <div className="hidden lg:flex w-full px-4 py-3 border-b border-border items-center">
        <div className="bg-primary/10 text-primary text-xs px-2 py-1 rounded-md font-medium tracking-wide w-fit">
          {t("Role")}
        </div>
      </div>

      <nav className="flex-1 w-full py-4 flex flex-col gap-1 px-2 lg:px-3 overflow-y-auto">
        {navItems.map((item) => {
          const isActive = location === item.href;
          return (
            <Link key={item.href} href={item.href}>
              <div 
                className={`flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors ${
                  isActive 
                    ? "bg-accent text-accent-foreground font-medium" 
                    : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                }`}
                title={t(item.label)}
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                <span className="hidden lg:block text-sm">{t(item.label)}</span>
              </div>
            </Link>
          );
        })}
      </nav>

      <div className="w-full border-t border-border p-2 lg:p-3 flex flex-col gap-1">
        {bottomItems.map((item) => {
          const isActive = location === item.href;
          return (
            <Link key={item.href} href={item.href}>
              <div 
                className={`flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer transition-colors ${
                  isActive 
                    ? "bg-accent text-accent-foreground font-medium" 
                    : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                }`}
                title={t(item.label)}
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                <span className="hidden lg:block text-sm">{t(item.label)}</span>
              </div>
            </Link>
          );
        })}
        
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <div className="flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer text-muted-foreground hover:bg-accent/50 hover:text-foreground mt-2" title="Language">
              <Languages className="w-5 h-5 flex-shrink-0" />
              <span className="hidden lg:block text-sm">English / عربي</span>
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" side="right">
            <DropdownMenuItem onClick={() => changeLanguage("en")}>English</DropdownMenuItem>
            <DropdownMenuItem onClick={() => changeLanguage("ar")}>عربي (Arabic)</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <div className="flex items-center gap-3 px-2 py-3 mt-2 rounded-md cursor-pointer text-muted-foreground hover:bg-accent/50 hover:text-foreground w-full">
          <Avatar className="w-8 h-8 rounded-md border border-border">
            <AvatarFallback className="bg-background text-xs rounded-md">OP</AvatarFallback>
          </Avatar>
          <div className="hidden lg:flex flex-col">
            <span className="text-sm font-medium text-foreground leading-none">Operator</span>
            <span className="text-xs text-muted-foreground mt-1">Admin</span>
          </div>
        </div>
      </div>
    </div>
  );
}