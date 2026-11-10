/** Built-in department display names; custom names and submitted values stay unchanged. */
const DEPARTMENT_LABELS: Readonly<Record<string, string>> = {
  IT部门: 'IT',
  财务部门: 'Finance',
  人事部门: 'HR',
  市场部门: 'Marketing',
};

export const departmentLabel = (value?: string | null): string =>
  value ? (Object.hasOwn(DEPARTMENT_LABELS, value) ? DEPARTMENT_LABELS[value] : value) : '';
