'use client';

import React, { useState } from 'react';
import { ThumbsUp, ThumbsDown, Star, MessageSquare } from 'lucide-react';
import { aiSaveFeedback, AIFeedbackRequest } from '@/lib/api/ai-api';

interface AIFeedbackProps {
  kind: string;
  query?: string;
  itemType?: string;
  itemId?: number;
  onFeedbackSubmitted?: () => void;
  className?: string;
}

export const AIFeedback: React.FC<AIFeedbackProps> = ({
  kind,
  query,
  itemType,
  itemId,
  onFeedbackSubmitted,
  className = '',
}) => {
  const [feedback, setFeedback] = useState<'useful' | 'not-useful' | null>(null);
  const [score, setScore] = useState<number>(0);
  const [notes, setNotes] = useState('');
  const [showNotes, setShowNotes] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const handleFeedback = async (isUseful: boolean) => {
    if (submitted) return;

    setFeedback(isUseful ? 'useful' : 'not-useful');

    // Auto-submit simple feedback
    await submitFeedback(isUseful);
  };

  const handleScoreChange = (newScore: number) => {
    setScore(newScore);
  };

  const handleNotesChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setNotes(e.target.value);
  };

  const submitFeedback = async (isUseful: boolean, includeScore = false, includeNotes = false) => {
    if (submitted) return;

    setSubmitting(true);
    try {
      const feedbackData: AIFeedbackRequest = {
        kind,
        query,
        item_type: itemType,
        item_id: itemId,
        useful: isUseful,
        ...(includeScore && score > 0 && { score }),
        ...(includeNotes && notes.trim() && { notes: notes.trim() }),
      };

      await aiSaveFeedback(feedbackData);
      setSubmitted(true);
      onFeedbackSubmitted?.();
    } catch (error) {
      console.error('Failed to submit feedback:', error);
      // Reset feedback state on error
      setFeedback(null);
      setScore(0);
      setNotes('');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDetailedFeedback = async () => {
    if (feedback === null) return;

    const isUseful = feedback === 'useful';
    await submitFeedback(isUseful, true, true);
  };

  if (submitted) {
    return (
      <div className={`text-sm text-green-600 flex items-center ${className}`}>
        <ThumbsUp className='w-4 h-4 mr-1' />
        感谢您的反馈！
      </div>
    );
  }

  return (
    <div className={`flex items-center space-x-2 ${className}`}>
      {/* Thumbs up/down buttons */}
      <button
        onClick={() => handleFeedback(true)}
        disabled={submitting}
        className={`p-1 rounded transition-colors ${
          feedback === 'useful'
            ? 'text-green-600 bg-green-100'
            : 'text-gray-400 hover:text-green-600 hover:bg-green-50'
        }`}
        title='这个建议有用'
      >
        <ThumbsUp className='w-4 h-4' />
      </button>

      <button
        onClick={() => handleFeedback(false)}
        disabled={submitting}
        className={`p-1 rounded transition-colors ${
          feedback === 'not-useful'
            ? 'text-red-600 bg-red-100'
            : 'text-gray-400 hover:text-red-600 hover:bg-red-50'
        }`}
        title='这个建议没用'
      >
        <ThumbsDown className='w-4 h-4' />
      </button>

      {/* Score selector (only show if feedback is useful) */}
      {feedback === 'useful' && (
        <div className='flex items-center space-x-1 ml-2'>
          {[1, 2, 3, 4, 5].map(starScore => (
            <button
              key={starScore}
              onClick={() => handleScoreChange(starScore)}
              disabled={submitting}
              className={`p-1 transition-colors ${
                score >= starScore ? 'text-yellow-500' : 'text-gray-300 hover:text-yellow-400'
              }`}
              title={`评分 ${starScore}`}
            >
              <Star className='w-3 h-3' />
            </button>
          ))}
        </div>
      )}

      {/* Notes button */}
      {feedback && (
        <button
          onClick={() => setShowNotes(!showNotes)}
          disabled={submitting}
          className='p-1 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors'
          title='添加备注'
        >
          <MessageSquare className='w-4 h-4' />
        </button>
      )}

      {/* Notes input */}
      {showNotes && (
        <div className='ml-2'>
          <textarea
            value={notes}
            onChange={handleNotesChange}
            placeholder='添加备注（可选）'
            className='w-32 h-16 px-2 py-1 text-xs border border-gray-300 rounded resize-none focus:outline-none focus:ring-1 focus:ring-blue-500'
            disabled={submitting}
          />
          <div className='flex space-x-1 mt-1'>
            <button
              onClick={handleDetailedFeedback}
              disabled={submitting}
              className='px-2 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50'
            >
              {submitting ? '提交中...' : '提交'}
            </button>
            <button
              onClick={() => setShowNotes(false)}
              disabled={submitting}
              className='px-2 py-1 text-xs border border-gray-300 text-gray-600 rounded hover:bg-gray-50 disabled:opacity-50'
            >
              取消
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
