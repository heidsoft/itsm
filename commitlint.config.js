module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',     // New feature
        'fix',      // Bug fix
        'docs',     // Documentation only
        'style',    // Formatting (no code change)
        'refactor', // Code restructuring
        'perf',     // Performance improvement
        'test',     // Adding tests
        'build',    // Build system changes
        'ci',       // CI/CD changes
        'chore',    // Maintenance
        'revert',   // Revert previous commit
      ],
    ],
    'subject-case': [2, 'never', ['sentence-case', 'start-case', 'pascal-case', 'upper-case']],
  },
};