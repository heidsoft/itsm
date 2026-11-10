'use client';

import { useEffect } from 'react';
import dayjs from 'dayjs';
import 'dayjs/locale/en';

export const DayjsLocaleSync: React.FC = () => {
  useEffect(() => { dayjs.locale('en'); }, []);
  return null;
};
