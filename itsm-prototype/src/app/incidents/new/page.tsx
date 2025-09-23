'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Search, X } from 'lucide-react';
import { FormInput } from '../../components/FormInput';
import { FormTextarea } from '../../components/FormTextarea';

interface ConfigItem {
  id: string;
  name: string;
  type: string;
}

export default function NewIncidentPage() {
  const router = useRouter();
  const [selectedCIs, setSelectedCIs] = useState<ConfigItem[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [searchResults, setSearchResults] = useState<ConfigItem[]>([]);

  // Mock data for configuration items
  const mockCIs: ConfigItem[] = [
    { id: '1', name: 'Web Server 01', type: 'Server' },
    { id: '2', name: 'Database Server', type: 'Database' },
    { id: '3', name: 'Load Balancer', type: 'Network' },
    { id: '4', name: 'Application Server', type: 'Server' },
  ];

  useEffect(() => {
    if (searchTerm.trim()) {
      // Simulate API call
      setTimeout(() => {
        const results = mockCIs.filter(ci => 
          ci.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
          ci.type.toLowerCase().includes(searchTerm.toLowerCase())
        );
        setSearchResults(results);
      }, 300);
    } else {
      setSearchResults([]);
    }
  }, [searchTerm]);

  const handleAddCI = (ci: ConfigItem) => {
    if (!selectedCIs.find(item => item.id === ci.id)) {
      setSelectedCIs([...selectedCIs, ci]);
    }
    setSearchTerm('');
  };

  const handleRemoveCI = (ciId: string) => {
    setSelectedCIs(selectedCIs.filter(ci => ci.id !== ciId));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const formData = new FormData(e.target as HTMLFormElement);
    
    const data = {
      title: formData.get('title') as string,
      description: formData.get('description') as string,
      priority: formData.get('priority') as string,
      type: formData.get('type') as string,
      source: 'manual',
      affected_cis: selectedCIs.map(ci => ci.id),
    };

    try {
      console.log('Creating new incident:', JSON.stringify(data, null, 2));
      alert('Incident created successfully!');
      router.push('/incidents'); // Redirect to incidents list after submission
    } catch (error) {
      console.error('Failed to create incident:', error);
      alert('Failed to create incident. Please try again.');
    }
  };

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
          <ArrowLeft className="w-5 h-5 mr-2" />
          Back
        </button>
        <h2 className="text-4xl font-bold text-gray-800">Create New Incident</h2>
        <p className="text-gray-500 mt-1">Manually record a new IT incident</p>
      </header>

      <div className="bg-white p-8 rounded-lg shadow-md">
        <form onSubmit={handleSubmit}>
          <div className="space-y-6">
            <FormInput label="Incident Title" id="title" name="title" type="text" required placeholder="Brief description of the incident" />
            <FormTextarea label="Detailed Description" id="description" name="description" rows={6} required placeholder="Please provide detailed information about the incident, including impact scope, occurrence time, etc..." />

            {/* Configuration Items Selection */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Affected Configuration Items
              </label>
              
              <div className="flex items-center justify-between p-3 bg-blue-50 border border-blue-200 rounded-lg">
                <div className="flex items-center space-x-2">
                  <Search className="w-4 h-4 text-blue-600" />
                  <input
                    type="text"
                    placeholder="Search configuration items..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="bg-transparent border-none outline-none flex-1 text-sm"
                  />
                </div>
              </div>

              {/* Search Results */}
              {searchResults.length > 0 && (
                <div className="mt-2 bg-white border border-gray-200 rounded-lg shadow-sm max-h-40 overflow-y-auto">
                  {searchResults.map((ci) => (
                    <div
                      key={ci.id}
                      onClick={() => handleAddCI(ci)}
                      className="p-3 hover:bg-gray-50 cursor-pointer border-b border-gray-100 last:border-b-0"
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <div className="font-medium text-sm">{ci.name}</div>
                          <div className="text-xs text-gray-500">{ci.type}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Selected CIs */}
              {selectedCIs.length > 0 && (
                <div className="mt-3">
                  <div className="text-sm font-medium text-gray-700 mb-2">Selected Items:</div>
                  <div className="flex flex-wrap gap-2">
                    {selectedCIs.map((ci) => (
                      <div key={ci.id} className="flex items-center bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm">
                        <span>{ci.name}</span>
                        <button
                          type="button"
                          onClick={() => handleRemoveCI(ci.id)}
                          className="ml-2 text-blue-600 hover:text-blue-800"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Priority Selection */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Priority</label>
              <select name="priority" required className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent">
                <option value="">Select Priority</option>
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="critical">Critical</option>
              </select>
            </div>

            {/* Type Selection */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Incident Type</label>
              <select name="type" required className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent">
                <option value="">Select Type</option>
                <option value="hardware">Hardware</option>
                <option value="software">Software</option>
                <option value="network">Network</option>
                <option value="security">Security</option>
                <option value="performance">Performance</option>
                <option value="other">Other</option>
              </select>
            </div>
          </div>

          <div className="mt-8 flex justify-end space-x-4">
            <button
              type="button"
              onClick={() => router.back()}
              className="px-6 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Create Incident
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}