import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const resources = {
  en: {
    translation: {
      "Command Center": "Command Center",
      "Queue": "Ranked Queue",
      "Trust": "Data Quality / Trust",
      "Governance": "Governance / Admin",
      "Executive": "Executive Summary",
      "About": "About",
      "Glossary": "Glossary",
      "Watchlists": "Watchlists",
      "Role": "Analyst",
      "Search...": "Search...",
      "Now": "Now",
      "1h": "1h",
      "24h": "24h",
      "7d": "7d",
      "Custom": "Custom",
      "Severity": "Severity",
      "Asset Class": "Asset Class",
      "Trust State": "Trust State",
      "All Systems Nominal": "All Systems Nominal",
      "Cache Health": "Cache Health",
      "Feed Health": "Feed Health"
    }
  },
  ar: {
    translation: {
      "Command Center": "مركز القيادة",
      "Queue": "قائمة الانتظار مرتبة",
      "Trust": "الثقة / جودة البيانات",
      "Governance": "الإدارة والحوكمة",
      "Executive": "الملخص التنفيذي",
      "About": "حول",
      "Glossary": "قاموس المصطلحات",
      "Watchlists": "قوائم المراقبة",
      "Role": "محلل",
      "Search...": "بحث...",
      "Now": "الآن",
      "1h": "١ ساعة",
      "24h": "٢٤ ساعة",
      "7d": "٧ أيام",
      "Custom": "مخصص",
      "Severity": "الخطورة",
      "Asset Class": "فئة الأصول",
      "Trust State": "حالة الثقة",
      "All Systems Nominal": "جميع الأنظمة تعمل بشكل طبيعي",
      "Cache Health": "صحة التخزين المؤقت",
      "Feed Health": "صحة التغذية"
    }
  }
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: "en",
    fallbackLng: "en",
    interpolation: {
      escapeValue: false 
    }
  });

export default i18n;