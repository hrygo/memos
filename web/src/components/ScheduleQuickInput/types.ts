/** Template for quick schedule creation */
export interface ScheduleTemplate {
  id: string;
  title: string;
  icon: string;
  duration: number; // in minutes
  defaultTitle?: string;
  color?: string;
  /** i18n key for translating the title */
  i18nKey?: string;
  /** Natural language prompt example (shown in input when selected) */
  prompt?: string;
  /** i18n key for the prompt */
  promptI18nKey?: string;
}
