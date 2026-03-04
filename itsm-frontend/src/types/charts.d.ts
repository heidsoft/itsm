declare module '@ant-design/charts' {
  import { FC } from 'react';

  export interface LineConfig {
    data?: unknown[];
    xField?: string;
    yField?: string;
    [key: string]: unknown;
  }

  export interface BarConfig {
    data?: unknown[];
    xField?: string;
    yField?: string;
    [key: string]: unknown;
  }

  export interface PieConfig {
    data?: unknown[];
    angleField?: string;
    colorField?: string;
    [key: string]: unknown;
  }

  export interface AreaConfig {
    data?: unknown[];
    xField?: string;
    yField?: string;
    [key: string]: unknown;
  }

  export interface ColumnConfig {
    data?: unknown[];
    xField?: string;
    yField?: string;
    [key: string]: unknown;
  }

  export interface DualAxesConfig {
    data?: unknown[] | any[][];
    xField?: string;
    yField?: string | string[];
    [key: string]: unknown;
  }

  export const Line: FC<LineConfig>;
  export const Bar: FC<BarConfig>;
  export const Pie: FC<PieConfig>;
  export const Area: FC<AreaConfig>;
  export const Column: FC<ColumnConfig>;
  export const DualAxes: FC<DualAxesConfig>;
}
