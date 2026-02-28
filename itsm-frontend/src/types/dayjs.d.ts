// Dayjs type declarations
import dayjs from 'dayjs';

declare module 'dayjs' {
  interface Dayjs {
    format(template?: string): string;
    add(value: number, unit?: dayjs.ManipulateType): Dayjs;
    subtract(value: number, unit?: dayjs.ManipulateType): Dayjs;
    diff(date?: dayjs.ConfigType, unit?: dayjs.QUnitType, float?: boolean): number;
    toNow(withoutSuffix?: boolean): string;
    fromNow(withoutSuffix?: boolean): string;
    startOf(unit: dayjs.OpUnitType): Dayjs;
    endOf(unit: dayjs.OpUnitType): Dayjs;
    isBefore(date?: dayjs.ConfigType, unit?: dayjs.OpUnitType): boolean;
    isAfter(date?: dayjs.ConfigType, unit?: dayjs.OpUnitType): boolean;
    isSame(date?: dayjs.ConfigType, unit?: dayjs.OpUnitType): boolean;
    toDate(): Date;
    toJSON(): string;
    unix(): number;
    daysInMonth(): number;
    toArray(): number[];
    toObject(): dayjs.ObjectType;
    valueOf(): number;
    clone(): Dayjs;
  }
}

export default dayjs;
