"use client";

import { useEffect, useRef, useState } from "react";
import Prism from "prismjs";
import "prismjs/components/prism-python";
import { Copy, Check } from "lucide-react";

interface CodeViewerProps {
  code: string;
  language?: string;
}

export function CodeViewer({ code, language = "python" }: CodeViewerProps) {
  const codeRef = useRef<HTMLElement>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (codeRef.current) {
      Prism.highlightElement(codeRef.current);
    }
  }, [code, language]);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const lines = code.split("\n");

  return (
    <div className="relative rounded-lg overflow-hidden border border-gray-700 bg-[#1d1f21]">
      <div className="flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
        <span className="text-xs text-gray-400 font-mono">{language}</span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-xs text-gray-400 hover:text-white transition-colors"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5" />
              복사됨
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              복사
            </>
          )}
        </button>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <tbody>
            {lines.map((line, i) => (
              <tr key={i} className="hover:bg-gray-800/50">
                <td className="px-4 py-0 text-right text-xs text-gray-600 select-none w-10 font-mono">
                  {i + 1}
                </td>
                <td className="px-4 py-0">
                  <pre className="!bg-transparent !p-0 !m-0 text-sm">
                    <code
                      ref={i === 0 ? codeRef : undefined}
                      className={`language-${language}`}
                      dangerouslySetInnerHTML={{
                        __html:
                          i === 0
                            ? Prism.highlight(
                                line,
                                Prism.languages[language] ?? Prism.languages.plain,
                                language
                              )
                            : Prism.highlight(
                                line,
                                Prism.languages[language] ?? Prism.languages.plain,
                                language
                              ),
                      }}
                    />
                  </pre>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
