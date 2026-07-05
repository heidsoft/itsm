/**
 * LoadingSpinner Component Tests
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { LoadingSpinner } from '../LoadingSpinner';

describe('LoadingSpinner', () => {
  it('renders without crashing', () => {
    render(<LoadingSpinner />);
    // Ant Design Loading uses role="status" with aria-label
    expect(screen.getByRole('status', { name: /加载中/i })).toBeInTheDocument();
  });

  it('displays loading indicator', () => {
    render(<LoadingSpinner />);
    // Check for the loading icon
    expect(screen.getByLabelText(/加载中/i)).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<LoadingSpinner className="custom-spinner" />);
    // The custom class should be applied to the spinner container
    const spinner = screen.getByRole('status', { name: /加载中/i });
    expect(spinner.closest('span')).toHaveClass('custom-spinner');
  });

  it('renders with custom size', () => {
    render(<LoadingSpinner size="sm" />);
    expect(screen.getByRole('status', { name: /加载中/i })).toBeInTheDocument();
  });

  it('renders with default large size', () => {
    render(<LoadingSpinner size="lg" />);
    expect(screen.getByRole('status', { name: /加载中/i })).toBeInTheDocument();
  });
});
