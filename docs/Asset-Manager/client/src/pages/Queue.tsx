import { useTranslation } from "react-i18next";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { 
  MoreHorizontal, 
  ArrowUpRight, 
  ShieldAlert, 
  Fingerprint,
  Clock,
  ExternalLink,
  ChevronRight
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const incidents = [
  {
    id: "INC-9041",
    symbol: "BTC/USD",
    type: "Spoofing / Layering",
    severity: "critical",
    riskScore: 94,
    timestamp: "2026-02-26 14:22:12",
    status: "new",
    region: "Global",
    evidenceCount: 14
  },
  {
    id: "INC-9040",
    symbol: "TSLA",
    type: "Wash Trading",
    severity: "high",
    riskScore: 88,
    timestamp: "2026-02-26 14:21:05",
    status: "triaged",
    region: "US-EAST",
    evidenceCount: 8
  },
  {
    id: "INC-9039",
    symbol: "ETH/EUR",
    type: "Momentum Ignition",
    severity: "elevated",
    riskScore: 72,
    timestamp: "2026-02-26 14:19:44",
    status: "new",
    region: "EU-WEST",
    evidenceCount: 5
  },
  {
    id: "INC-9038",
    symbol: "AAPL",
    type: "Anomalous Volatility",
    severity: "elevated",
    riskScore: 65,
    timestamp: "2026-02-26 14:10:32",
    status: "investigating",
    region: "US-EAST",
    evidenceCount: 12
  },
  {
    id: "INC-9037",
    symbol: "XAU/USD",
    type: "Quote Stuffing",
    severity: "high",
    riskScore: 81,
    timestamp: "2026-02-26 14:05:11",
    status: "new",
    region: "Global",
    evidenceCount: 22
  }
];

const getSeverityStyles = (severity: string) => {
  switch (severity) {
    case 'critical': return 'bg-[hsl(var(--critical)/0.15)] text-[hsl(var(--critical))] border-[hsl(var(--critical)/0.2)]';
    case 'high': return 'bg-[hsl(var(--high-risk)/0.15)] text-[hsl(var(--high-risk))] border-[hsl(var(--high-risk)/0.2)]';
    case 'elevated': return 'bg-[hsl(var(--elevated)/0.15)] text-[hsl(var(--elevated))] border-[hsl(var(--elevated)/0.2)]';
    default: return 'bg-muted text-muted-foreground border-border';
  }
};

const getStatusBadge = (status: string) => {
  switch (status) {
    case 'new': return <Badge variant="outline" className="bg-primary/5 text-primary border-primary/20 uppercase text-[10px] tracking-wider">New</Badge>;
    case 'triaged': return <Badge variant="outline" className="bg-blue-500/10 text-blue-500 border-blue-500/20 uppercase text-[10px] tracking-wider">Triaged</Badge>;
    case 'investigating': return <Badge variant="outline" className="bg-amber-500/10 text-amber-500 border-amber-500/20 uppercase text-[10px] tracking-wider">Investigating</Badge>;
    default: return null;
  }
};

export default function Queue() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-6 animate-in fade-in slide-in-from-bottom-2 duration-500 max-w-7xl mx-auto h-full pb-10">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <ShieldAlert className="w-5 h-5 text-muted-foreground" />
            <h1 className="text-2xl font-bold tracking-tight">{t("Queue")}</h1>
          </div>
          <p className="text-muted-foreground text-sm">Real-time ranked incidents requiring analyst disposition.</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" className="h-9 border-border/50">
            <Clock className="w-4 h-4 mr-2" />
            History
          </Button>
          <Button size="sm" className="h-9 bg-primary text-primary-foreground shadow-sm">
            Bulk Disposition
          </Button>
        </div>
      </div>

      <div className="border border-border rounded-xl bg-card overflow-hidden shadow-sm">
        <Table>
          <TableHeader className="bg-muted/30">
            <TableRow className="hover:bg-transparent">
              <TableHead className="w-[120px] font-mono text-xs uppercase tracking-wider py-4">Incident ID</TableHead>
              <TableHead className="font-mono text-xs uppercase tracking-wider">Symbol</TableHead>
              <TableHead className="font-mono text-xs uppercase tracking-wider">Risk Type</TableHead>
              <TableHead className="font-mono text-xs uppercase tracking-wider text-center">Score</TableHead>
              <TableHead className="font-mono text-xs uppercase tracking-wider">Severity</TableHead>
              <TableHead className="font-mono text-xs uppercase tracking-wider">Status</TableHead>
              <TableHead className="font-mono text-xs uppercase tracking-wider">Timestamp</TableHead>
              <TableHead className="w-[50px]"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {incidents.map((incident) => (
              <TableRow key={incident.id} className="cursor-pointer group hover:bg-accent/30 transition-colors">
                <TableCell className="font-mono text-sm font-semibold py-4">
                  {incident.id}
                </TableCell>
                <TableCell>
                  <div className="flex flex-col">
                    <span className="font-bold text-foreground">{incident.symbol}</span>
                    <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                      <Globe className="w-2.5 h-2.5" /> {incident.region}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Fingerprint className="w-4 h-4 text-muted-foreground opacity-50" />
                    <span className="text-sm">{incident.type}</span>
                  </div>
                </TableCell>
                <TableCell className="text-center">
                  <span className={`font-mono font-bold text-lg ${
                    incident.riskScore > 90 ? 'text-[hsl(var(--critical))]' :
                    incident.riskScore > 80 ? 'text-[hsl(var(--high-risk))]' : 'text-[hsl(var(--elevated))]'
                  }`}>
                    {incident.riskScore}
                  </span>
                </TableCell>
                <TableCell>
                  <Badge variant="outline" className={`font-semibold text-[10px] uppercase tracking-wider px-2 py-0 h-5 ${getSeverityStyles(incident.severity)}`}>
                    {incident.severity}
                  </Badge>
                </TableCell>
                <TableCell>
                  {getStatusBadge(incident.status)}
                </TableCell>
                <TableCell>
                  <div className="flex flex-col">
                    <span className="text-sm font-mono">{incident.timestamp.split(' ')[1]}</span>
                    <span className="text-[10px] text-muted-foreground">{incident.timestamp.split(' ')[0]}</span>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center justify-end">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem className="flex items-center gap-2">
                          <ArrowUpRight className="w-4 h-4" /> View Details
                        </DropdownMenuItem>
                        <DropdownMenuItem className="flex items-center gap-2">
                          <ExternalLink className="w-4 h-4" /> Replay Feed
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <ChevronRight className="w-4 h-4 text-muted-foreground/30 group-hover:text-primary transition-colors ml-1" />
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <div className="p-4 bg-muted/20 border-t border-border flex items-center justify-between text-xs text-muted-foreground">
          <div className="flex items-center gap-4">
            <span>Showing 5 of 128 active incidents</span>
            <div className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-[hsl(var(--stable))] animate-pulse"></span>
              Live updates enabled
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="h-7 text-xs px-2" disabled>Previous</Button>
            <Button variant="ghost" size="sm" className="h-7 text-xs px-2">Next</Button>
          </div>
        </div>
      </div>
    </div>
  );
}

function Globe({ className }: { className?: string }) {
  return (
    <svg 
      xmlns="http://www.w3.org/2000/svg" 
      width="24" 
      height="24" 
      viewBox="0 0 24 24" 
      fill="none" 
      stroke="currentColor" 
      strokeWidth="2" 
      strokeLinecap="round" 
      strokeLinejoin="round" 
      className={className}
    >
      <circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20"/><path d="M2 12h20"/>
    </svg>
  );
}