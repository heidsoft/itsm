import React, { useState, useEffect, useRef, useCallback } from "react";
import { Search, Clock, FileText, User, Database, Ticket } from "lucide-react";
import { httpClient } from "../lib/http-client";
import { useRouter } from "next/navigation";

interface SearchResult {
  id: string;
  title: string;
  type: "ticket" | "knowledge" | "user" | "ci" | "service";
  description?: string;
  url: string;
  highlight?: string;
}

interface GlobalSearchProps {
  placeholder?: string;
  className?: string;
}

const typeIcons = {
  ticket: Ticket,
  knowledge: FileText,
  user: User,
  ci: Database,
  service: FileText,
};

const typeLabels = {
  ticket: "工单",
  knowledge: "知识库",
  user: "用户",
  ci: "配置项",
  service: "服务",
};

export const GlobalSearch: React.FC<GlobalSearchProps> = ({
  placeholder = "搜索工单、知识库、用户...",
  className = "",
}) => {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [showResults, setShowResults] = useState(false);
  const [recentSearches, setRecentSearches] = useState<string[]>([]);
  const searchRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  // 防抖搜索
  const debouncedSearch = useCallback(
    debounce(async (searchQuery: string) => {
      if (!searchQuery.trim()) {
        setResults([]);
        return;
      }

      setLoading(true);
      try {
        const response = await httpClient.get<SearchResult[]>("/api/search", {
          q: searchQuery,
          limit: 10,
        });
        setResults(response.data || []);
      } catch (error) {
        console.error("搜索失败:", error);
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 300),
    []
  );

  useEffect(() => {
    debouncedSearch(query);
  }, [query, debouncedSearch]);

  // 点击外部关闭搜索结果
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        searchRef.current &&
        !searchRef.current.contains(event.target as Node)
      ) {
        setShowResults(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // 加载最近搜索
  useEffect(() => {
    const saved = localStorage.getItem("recent_searches");
    if (saved) {
      setRecentSearches(JSON.parse(saved));
    }
  }, []);

  const handleSearch = (searchQuery: string) => {
    if (!searchQuery.trim()) return;

    // 保存到最近搜索
    const updated = [
      searchQuery,
      ...recentSearches.filter((s) => s !== searchQuery),
    ].slice(0, 5);
    setRecentSearches(updated);
    localStorage.setItem("recent_searches", JSON.stringify(updated));

    setQuery(searchQuery);
    setShowResults(true);
  };

  const handleResultClick = (result: SearchResult) => {
    setShowResults(false);
    setQuery("");
    router.push(result.url);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (results.length > 0) {
        handleResultClick(results[0]);
      }
    } else if (e.key === "Escape") {
      setShowResults(false);
      inputRef.current?.blur();
    }
  };

  return (
    <div ref={searchRef} className={`relative ${className}`}>
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => setShowResults(true)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="w-full pl-10 pr-4 py-3 bg-white border border-gray-300 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200 shadow-sm"
        />
        {loading && (
          <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
          </div>
        )}
      </div>

      {/* 搜索结果下拉框 */}
      {showResults && (
        <div className="absolute top-full left-0 right-0 mt-2 bg-white border border-gray-200 rounded-xl shadow-lg z-50 max-h-96 overflow-y-auto">
          {query.trim() === "" ? (
            // 显示最近搜索
            <div className="p-4">
              <h3 className="text-sm font-medium text-gray-700 mb-3 flex items-center">
                <Clock className="w-4 h-4 mr-2" />
                最近搜索
              </h3>
              {recentSearches.length > 0 ? (
                <div className="space-y-2">
                  {recentSearches.map((search, index) => (
                    <button
                      key={index}
                      onClick={() => handleSearch(search)}
                      className="w-full text-left px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 rounded-lg transition-colors"
                    >
                      {search}
                    </button>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">暂无最近搜索</p>
              )}
            </div>
          ) : results.length > 0 ? (
            // 显示搜索结果
            <div className="py-2">
              {results.map((result) => {
                const Icon = typeIcons[result.type];
                return (
                  <button
                    key={result.id}
                    onClick={() => handleResultClick(result)}
                    className="w-full px-4 py-3 text-left hover:bg-gray-50 transition-colors border-b border-gray-100 last:border-b-0"
                  >
                    <div className="flex items-start space-x-3">
                      <div className="flex-shrink-0 mt-1">
                        <Icon className="w-4 h-4 text-gray-400" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center space-x-2">
                          <span className="text-sm font-medium text-gray-900 truncate">
                            {result.title}
                          </span>
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">
                            {typeLabels[result.type]}
                          </span>
                        </div>
                        {result.description && (
                          <p className="text-sm text-gray-600 truncate mt-1">
                            {result.description}
                          </p>
                        )}
                        {result.highlight && (
                          <p
                            className="text-xs text-blue-600 mt-1"
                            dangerouslySetInnerHTML={{
                              __html: result.highlight,
                            }}
                          />
                        )}
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          ) : query.trim() !== "" && !loading ? (
            // 无搜索结果
            <div className="p-4 text-center">
              <p className="text-sm text-gray-500">未找到相关结果</p>
            </div>
          ) : null}
        </div>
      )}
    </div>
  );
};

// 防抖函数
function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout;
  return (...args: Parameters<T>) => {
    clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}
