import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { 
  AlertCircle, 
  Target, 
  RefreshCw, 
  BarChart2, 
  Search, 
  Zap, 
  ShieldCheck, 
  Layers,
  History,
  ArrowRight,
  TrendingUp,
  Activity
} from "lucide-react";
import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  Tooltip, 
  ResponsiveContainer, 
  CartesianGrid, 
  LineChart, 
  Line, 
  AreaChart, 
  Area 
} from "recharts";
import { motion, AnimatePresence } from "framer-motion";

const mockQueueData = [
  { id: "INC-9041", symbol: "BTC/USD", type: "Spoofing", severity: "critical", risk: 94, time: "12s ago" },
  { id: "INC-9040", symbol: "TSLA", type: "Wash Trade", severity: "high", risk: 88, time: "1m ago" },
  { id: "INC-9039", symbol: "ETH/EUR", type: "Momentum", severity: "elevated", risk: 72, time: "3m ago" },
  { id: "INC-9038", symbol: "AAPL", type: "Anomalous Vol", severity: "elevated", risk: 65, time: "12m ago" },
];

const activityData = [
  { time: '10:00', value: 45, critical: 5 },
  { time: '10:05', value: 30, critical: 2 },
  { time: '10:10', value: 65, critical: 8 },
  { time: '10:15', value: 40, critical: 3 },
  { time: '10:20', value: 85, critical: 15 },
  { time: '10:25', value: 50, critical: 6 },
  { time: '10:30', value: 95, critical: 22 },
];

const performanceData = [
  { name: '10s', latency: 4, drift: 2 },
  { name: '20s', latency: 5, drift: 3 },
  { name: '30s', latency: 3, drift: 1 },
  { name: '40s', latency: 7, drift: 4 },
  { name: '50s', latency: 4, drift: 2 },
  { name: '60s', latency: 5, drift: 3 },
];

export default function CommandCenter() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-6 max-w-[1600px] mx-auto pb-12">
      <motion.div 
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex flex-col md:flex-row md:items-center justify-between gap-4"
      >
        <div>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 rounded-xl">
              <Layers className="w-6 h-6 text-primary" />
            </div>
            <div>
              <h1 className="text-3xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-foreground to-foreground/70">
                {t("Command Center")}
              </h1>
              <p className="text-muted-foreground text-sm flex items-center gap-2 mt-0.5">
                <Activity className="w-3.5 h-3.5 text-stable" />
                System Status: <span className="text-stable font-semibold uppercase tracking-widest text-[10px]">Operational</span>
              </p>
            </div>
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          <div className="flex flex-col items-end px-4 border-r border-border/50 hidden lg:flex">
            <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">Processed Events</span>
            <span className="text-lg font-mono font-bold tabular-nums">14,284,902</span>
          </div>
          <div className="flex items-center gap-2 bg-card/80 backdrop-blur-xl border border-border/50 rounded-2xl p-1.5 shadow-2xl">
            <button className="px-4 py-1.5 rounded-xl bg-primary text-primary-foreground text-xs font-bold shadow-lg shadow-primary/20 transition-all hover:scale-105 active:scale-95">
              Live View
            </button>
            <button className="px-4 py-1.5 rounded-xl hover:bg-accent text-muted-foreground text-xs font-bold transition-all">
              Historical
            </button>
          </div>
        </div>
      </motion.div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {[
          { label: "Active Critical", value: "12", sub: "+3 (1h)", color: "text-[hsl(var(--critical))]", icon: AlertCircle },
          { label: "Data Quality", value: "99.98%", sub: "Optimal", color: "text-[hsl(var(--stable))]", icon: ShieldCheck },
          { label: "Avg Latency", value: "4.2ms", sub: "Network", color: "text-blue-500", icon: Zap },
          { label: "Risk Radius", value: "Level 4", sub: "Global", color: "text-[hsl(var(--high-risk))]", icon: Target },
        ].map((stat, i) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: i * 0.1 }}
          >
            <Card className="relative overflow-hidden group border-none shadow-2xl bg-gradient-to-br from-card to-card/50 hover:to-accent/10 transition-all duration-500">
              <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                <stat.icon className="w-12 h-12" />
              </div>
              <CardContent className="p-6">
                <div className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-2 flex items-center gap-2">
                  <span className={`w-1 h-4 rounded-full bg-current ${stat.color}`} />
                  {stat.label}
                </div>
                <div className="flex items-baseline gap-3">
                  <div className={`text-4xl font-black font-mono tracking-tighter ${stat.color}`}>
                    {stat.value}
                  </div>
                </div>
                <div className="mt-3 flex items-center gap-2 text-[10px] font-bold text-muted-foreground/60">
                  <TrendingUp className="w-3 h-3 text-stable" />
                  {stat.sub.toUpperCase()}
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <motion.div 
          className="lg:col-span-8"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.4 }}
        >
          <Card className="h-full border-none shadow-2xl bg-card/30 backdrop-blur-2xl overflow-hidden">
            <CardHeader className="flex flex-row items-center justify-between pb-8">
              <div>
                <CardTitle className="text-sm font-black uppercase tracking-[0.2em] text-muted-foreground flex items-center gap-2">
                  <BarChart2 className="w-4 h-4 text-primary" /> Risk Topology Feed
                </CardTitle>
                <p className="text-[10px] text-muted-foreground/60 mt-1 font-bold">MULTI-SOURCE STREAM AGGREGATION</p>
              </div>
              <div className="flex gap-2">
                <div className="h-1.5 w-8 rounded-full bg-primary/20 overflow-hidden">
                  <motion.div 
                    className="h-full bg-primary"
                    animate={{ x: ["-100%", "100%"] }}
                    transition={{ repeat: Infinity, duration: 2, ease: "linear" }}
                  />
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="h-[400px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={activityData} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
                    <defs>
                      <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0}/>
                      </linearGradient>
                      <linearGradient id="colorCritical" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="hsl(var(--critical))" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="hsl(var(--critical))" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border))" opacity={0.1} />
                    <XAxis dataKey="time" axisLine={false} tickLine={false} tick={{ fontSize: 10, fontWeight: 700, fill: 'hsl(var(--muted-foreground))' }} dy={10} />
                    <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 10, fontWeight: 700, fill: 'hsl(var(--muted-foreground))' }} />
                    <Tooltip 
                      contentStyle={{ backgroundColor: 'rgba(0,0,0,0.8)', border: 'none', borderRadius: '12px', backdropFilter: 'blur(10px)', color: 'white', fontSize: '10px' }}
                    />
                    <Area type="monotone" dataKey="value" stroke="hsl(var(--primary))" strokeWidth={3} fillOpacity={1} fill="url(#colorValue)" />
                    <Area type="monotone" dataKey="critical" stroke="hsl(var(--critical))" strokeWidth={2} fillOpacity={1} fill="url(#colorCritical)" strokeDasharray="5 5" />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
              <div className="grid grid-cols-3 gap-4 mt-8 pt-8 border-t border-border/10">
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Throughput</span>
                  <span className="text-xl font-mono font-black">18.4 GB/s</span>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Model Drift</span>
                  <span className="text-xl font-mono font-black text-stable">0.02%</span>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Uptime</span>
                  <span className="text-xl font-mono font-black">99.999%</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div 
          className="lg:col-span-4 flex flex-col gap-6"
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.5 }}
        >
          <Card className="border-none shadow-2xl bg-card/30 backdrop-blur-2xl flex flex-col flex-1">
            <CardHeader className="flex flex-row items-center justify-between pb-6">
              <CardTitle className="text-sm font-black uppercase tracking-[0.2em] text-muted-foreground flex items-center gap-2">
                <Target className="w-4 h-4 text-[hsl(var(--critical))]" /> Live Evidence
              </CardTitle>
              <button className="text-[10px] font-black hover:text-primary transition-colors uppercase tracking-widest">Refresh</button>
            </CardHeader>
            <CardContent className="flex-1 overflow-hidden p-0">
              <div className="flex flex-col px-1">
                {mockQueueData.map((item, i) => (
                  <motion.div 
                    key={item.id} 
                    className="mx-2 mb-2 p-4 rounded-2xl hover:bg-white/5 transition-all cursor-pointer group relative overflow-hidden"
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    <div className="flex justify-between items-start mb-3">
                      <div className="flex items-center gap-3">
                        <div className={`w-2.5 h-2.5 rounded-full shadow-[0_0_10px_rgba(0,0,0,0.5)] ${
                          item.severity === 'critical' ? 'bg-[hsl(var(--critical))] shadow-[hsl(var(--critical)/0.5)]' :
                          item.severity === 'high' ? 'bg-[hsl(var(--high-risk))]' : 'bg-[hsl(var(--elevated))]'
                        }`} />
                        <span className="font-mono text-xs font-black tracking-wider group-hover:text-primary transition-colors">{item.id}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">{item.time}</span>
                        <ArrowRight className="w-3 h-3 text-muted-foreground opacity-0 group-hover:opacity-100 -translate-x-2 group-hover:translate-x-0 transition-all" />
                      </div>
                    </div>
                    <div className="flex justify-between items-end">
                      <div>
                        <div className="text-lg font-black tracking-tight">{item.symbol}</div>
                        <div className="text-[10px] font-black text-muted-foreground uppercase tracking-widest">{item.type}</div>
                      </div>
                      <div className="flex flex-col items-end">
                        <div className={`text-2xl font-black font-mono tracking-tighter ${
                          item.risk > 90 ? 'text-[hsl(var(--critical))]' :
                          item.risk > 80 ? 'text-[hsl(var(--high-risk))]' : 'text-[hsl(var(--elevated))]'
                        }`}>
                          {item.risk}
                        </div>
                        <div className="text-[8px] font-black text-muted-foreground/40 uppercase tracking-widest leading-none">Risk Index</div>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            </CardContent>
            <div className="p-4 border-t border-border/10">
              <button className="w-full py-3 rounded-xl bg-accent/30 hover:bg-accent/50 text-[10px] font-black uppercase tracking-[0.2em] transition-all">
                Access Full Ranked Queue
              </button>
            </div>
          </Card>
        </motion.div>
      </div>

      <motion.div 
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.6 }}
        className="grid grid-cols-1 md:grid-cols-3 gap-6"
      >
        <Card className="border-none shadow-2xl bg-gradient-to-br from-blue-500/10 to-transparent p-6">
          <div className="flex items-center gap-3 mb-4">
            <Zap className="w-5 h-5 text-blue-500" />
            <span className="text-xs font-black uppercase tracking-[0.2em]">Network Pulse</span>
          </div>
          <div className="h-[100px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={performanceData}>
                <Line type="monotone" dataKey="latency" stroke="#3b82f6" strokeWidth={3} dot={false} />
                <Line type="monotone" dataKey="drift" stroke="#3b82f6" strokeWidth={1} strokeDasharray="3 3" dot={false} opacity={0.5} />
              </LineChart>
            </ResponsiveContainer>
          </div>
          <div className="mt-4 flex justify-between items-end">
             <span className="text-[10px] font-black text-muted-foreground uppercase tracking-widest">Edge Response</span>
             <span className="text-sm font-mono font-black text-blue-500">OPTIMAL</span>
          </div>
        </Card>

        <Card className="border-none shadow-2xl bg-gradient-to-br from-[hsl(var(--stable)/0.1)] to-transparent p-6">
          <div className="flex items-center gap-3 mb-4">
            <ShieldCheck className="w-5 h-5 text-[hsl(var(--stable))]" />
            <span className="text-xs font-black uppercase tracking-[0.2em]">Validation Chain</span>
          </div>
          <div className="flex flex-col gap-3">
            {[1, 2, 3].map(i => (
              <div key={i} className="flex items-center justify-between text-[10px] font-bold">
                <span className="text-muted-foreground">NODE_SENTINEL_0{i}</span>
                <span className="text-stable flex items-center gap-1">
                  <div className="w-1.5 h-1.5 rounded-full bg-stable" /> VERIFIED
                </span>
              </div>
            ))}
          </div>
        </Card>

        <Card className="border-none shadow-2xl bg-card/30 p-6 flex items-center justify-center group cursor-pointer overflow-hidden relative">
          <div className="absolute inset-0 bg-primary/5 opacity-0 group-hover:opacity-100 transition-opacity" />
          <div className="flex flex-col items-center gap-3 relative z-10">
            <History className="w-8 h-8 text-muted-foreground group-hover:text-primary transition-colors" />
            <span className="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground group-hover:text-foreground transition-colors">Audit Exports</span>
          </div>
        </Card>
      </motion.div>
    </div>
  );
}