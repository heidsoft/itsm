declare module '@ant-design/charts' {
  import { FC } from 'react';

  export interface LineConfig {
    data?: any[];
    xField?: string;
    yField?: string;
    [key: string]: any;
  }

  export interface BarConfig {
    data?: any[];
    xField?: string;
    yField?: string;
    [key: string]: any;
  }

  export interface PieConfig {
    data?: any[];
    angleField?: string;
    colorField?: string;
    [key: string]: any;
  }

  export interface AreaConfig {
    data?: any[];
    xField?: string;
    yField?: string;
    [key: string]: any;
  }

  export interface ColumnConfig {
    data?: any[];
    xField?: string;
    yField?: string;
    [key: string]: any;
  }

  export interface DualAxesConfig {
    data?: any[] | any[][];
    xField?: string;
    yField?: string | string[];
    [key: string]: any;
  }

  export const Line: FC<LineConfig>;
  export const Bar: FC<BarConfig>;
  export const Pie: FC<PieConfig>;
  export const Area: FC<AreaConfig>;
  export const Column: FC<ColumnConfig>;
  export const DualAxes: FC<DualAxesConfig>;
}
