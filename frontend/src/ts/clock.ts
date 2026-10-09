import { format } from "date-fns/format";
import type { Locale } from "date-fns/locale";
import { locales } from "./locales";

const CLOCK_UPDATE_INTERVAL = 5000;
const CLOCK_UPDATE_INTERVAL_SECONDS = 1000;

const TIME_FORMATS = {
    TWELVE_HOUR: "h:mma" as const,
    TWELVE_HOUR_SECONDS: "h:mm:ssa" as const,
    TWELVE_HOUR_NO_AMPM: "h:mm" as const,
    TWELVE_HOUR_SECONDS_NO_AMPM: "h:mm:ss" as const,
    TWENTY_FOUR_HOUR: "HH:mm" as const,
    TWENTY_FOUR_HOUR_SECONDS: "HH:mm:ss" as const,
} as const;

type TimeFormat = "12" | "24";

interface ClockConfig {
    showDate: boolean;
    dateFormat: string;
    showTime: boolean;
    timeFormat: TimeFormat;
    showSeconds: boolean;
    showAmPm: boolean;
    langCode: string;
    secondaryTimezone: string;
}

interface ClockElements {
    main: HTMLElement | null;
    date: HTMLElement | null;
    time: HTMLElement | null;
    secondaryTime: HTMLElement | null;
    secondaryDay: HTMLElement | null;
}

class Clock {
    private config: ClockConfig;
    private elements: ClockElements;
    private intervalId?: number;
    private langCode: string;
    private lang: Locale;

    constructor(config: ClockConfig) {
        this.config = config;
        this.elements = this.initialiseElements();
        this.langCode = this.config.langCode;
        this.lang = this.initialiseLang();
    }

    private initialiseLang(): Locale {
        const DEFAULT_LANG = "enGB";

        if (!this.langCode) {
            console.error(
                `No language code provided, defaulting to ${DEFAULT_LANG}`,
            );
            return locales[DEFAULT_LANG];
        }

        // Check for exact match
        if (locales[this.langCode]) {
            console.log(`Using language ${this.langCode}`);
            return locales[this.langCode];
        }

        const splitCode = this.langCode.split("_");

        if (splitCode.length > 1) {
            const joinedCode = splitCode[0] + splitCode[1];
            if (locales[joinedCode]) {
                console.log(`Using base language ${joinedCode}`);
                return locales[joinedCode];
            }
        }

        // Try base language code without region
        const baseCode = splitCode[0];
        if (locales[baseCode]) {
            console.log(`Using base language ${baseCode}`);
            return locales[baseCode];
        }

        // Fallback to default
        console.error(
            `Language ${this.langCode} not found, defaulting to ${DEFAULT_LANG}`,
        );

        return locales[DEFAULT_LANG];
    }

    private initialiseElements(): ClockElements {
        const main = document.getElementById("clock");
        if (!main) {
            console.warn(
                "Clock element not found - this is expected if UI is disabled",
            );
            return {
                main: null,
                date: null,
                time: null,
                secondaryTime: null,
                secondaryDay: null,
            };
        }
        return {
            main: main as HTMLElement,
            date: document.querySelector(".clock--date"),
            time: document.querySelector(".clock--time"),
            secondaryTime: document.querySelector(".clock--secondary-time"),
            secondaryDay: document.querySelector(".clock--secondary-day"),
        };
    }

    private updateDate(now: Date): void {
        if (!this.config.showDate || !this.elements.date) return;
        try {
            this.elements.date.innerHTML = format(now, this.config.dateFormat, {
                locale: this.lang,
            });
        } catch (error) {
            console.error("Error formatting date:", error);
            this.elements.date.innerHTML = now.toLocaleDateString();
        }
    }

    private timeFormat(): string {
        let timeFormat: string;
        if (this.config.timeFormat === "12") {
            if (this.config.showSeconds) {
                timeFormat = this.config.showAmPm
                    ? TIME_FORMATS.TWELVE_HOUR_SECONDS
                    : TIME_FORMATS.TWELVE_HOUR_SECONDS_NO_AMPM;
            } else {
                timeFormat = this.config.showAmPm
                    ? TIME_FORMATS.TWELVE_HOUR
                    : TIME_FORMATS.TWELVE_HOUR_NO_AMPM;
            }
        } else {
            timeFormat = this.config.showSeconds
                ? TIME_FORMATS.TWENTY_FOUR_HOUR_SECONDS
                : TIME_FORMATS.TWENTY_FOUR_HOUR;
        }
        return timeFormat;
    }

    private updateTime(now: Date): void {
        if (!this.config.showTime || !this.elements.time) return;

        const timeFormat = this.timeFormat();

        try {
            const formattedTime = format(now, timeFormat, {
                locale: this.lang,
            });
            this.elements.time.innerHTML =
                this.config.timeFormat === "12"
                    ? formattedTime.toLowerCase()
                    : formattedTime;
        } catch (error) {
            console.error("Error formatting time:", error);
            this.elements.time.innerHTML = now.toLocaleTimeString();
        }
    }

    /**
     * Wall-clock time in another IANA timezone as a local Date (so date-fns formats it
     * like the main clock), plus the day difference with the local date.
     */
    private wallClockIn(
        now: Date,
        timeZone: string,
    ): { wall: Date; days: number } {
        // "MM/DD/YYYY, HH:mm:ss" in the target zone; hour12:false may yield "24" at midnight
        const text = new Intl.DateTimeFormat("en-US", {
            timeZone,
            hour12: false,
            year: "numeric",
            month: "2-digit",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
        }).format(now);
        const m = text.match(/(\d+)\/(\d+)\/(\d+),?\s+(\d+):(\d+):(\d+)/);
        if (!m) {
            throw new Error(`Unexpected date format: ${text}`);
        }
        const [, month, day, year, hour, minute, second] = m.map(Number);
        const wall = new Date(year, month - 1, day, hour % 24, minute, second);
        const dayStart = (d: Date): number =>
            new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
        const days = Math.round((dayStart(wall) - dayStart(now)) / 86_400_000);
        return { wall, days };
    }

    private updateSecondaryTime(now: Date): void {
        const timeZone = this.config.secondaryTimezone;
        if (
            !this.config.showTime ||
            !timeZone ||
            !this.elements.secondaryTime
        ) {
            return;
        }
        try {
            const { wall, days } = this.wallClockIn(now, timeZone);
            const formatted = format(wall, this.timeFormat(), {
                locale: this.lang,
            });
            this.elements.secondaryTime.innerHTML =
                this.config.timeFormat === "12"
                    ? formatted.toLowerCase()
                    : formatted;
            if (this.elements.secondaryDay) {
                this.elements.secondaryDay.innerHTML =
                    days > 0 ? `+${days}` : days < 0 ? `${days}` : "";
            }
        } catch (error) {
            console.error("Error formatting secondary time:", error);
        }
    }

    private render(): void {
        const now = new Date();
        this.updateDate(now);
        this.updateTime(now);
        this.updateSecondaryTime(now);
    }

    public start(): void {
        if (this.intervalId) {
            this.stop();
        }
        this.render();
        this.intervalId = window.setInterval(
            () => this.render(),
            this.config.showTime && this.config.showSeconds
                ? CLOCK_UPDATE_INTERVAL_SECONDS
                : CLOCK_UPDATE_INTERVAL,
        );
    }

    public stop(): void {
        if (this.intervalId) {
            window.clearInterval(this.intervalId);
            this.intervalId = undefined;
        }
    }
}

function initClock(
    kioskShowDate: boolean,
    kioskDateFormat: string,
    kioskShowTime: boolean,
    kioskTimeFormat: TimeFormat,
    kioskShowSeconds: boolean,
    kioskShowAmPm: boolean,
    kioskLangCode: string,
    kioskSecondaryTimezone = "",
): Clock {
    const config: ClockConfig = {
        showDate: kioskShowDate,
        dateFormat: kioskDateFormat,
        showTime: kioskShowTime,
        timeFormat: kioskTimeFormat,
        showSeconds: kioskShowSeconds,
        showAmPm: kioskShowAmPm,
        langCode: kioskLangCode,
        secondaryTimezone: kioskSecondaryTimezone,
    };

    const clock = new Clock(config);
    clock.start();

    const handleUnload = () => clock.stop();
    window.addEventListener("unload", handleUnload);

    return Object.assign(clock, {
        cleanup: () => window.removeEventListener("unload", handleUnload),
    });
}

export { Clock, initClock, type TimeFormat };
