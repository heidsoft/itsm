/**
 * SLA 相关常量
 */

export const SLAPriority = {
    P1: 'P1',
    P2: 'P2',
    P3: 'P3',
    P4: 'P4',
};

export const SLAPriorityLabels = {
    [SLAPriority.P1]: '极高 (P1)',
    [SLAPriority.P2]: '高 (P2)',
    [SLAPriority.P3]: '中 (P3)',
    [SLAPriority.P4]: '低 (P4)',
};

export const SLAPriorityColors = {
    [SLAPriority.P1]: 'red',
    [SLAPriority.P2]: 'orange',
    [SLAPriority.P3]: 'blue',
    [SLAPriority.P4]: 'green',
};

export const SLAAlertLevel = {
    WARNING: 'warning',
    CRITICAL: 'critical',
    SEVERE: 'severe',
};

export const SLAAlertLevelLabels = {
    [SLAAlertLevel.WARNING]: '警告',
    [SLAAlertLevel.CRITICAL]: '严重',
    [SLAAlertLevel.SEVERE]: '危急',
};
