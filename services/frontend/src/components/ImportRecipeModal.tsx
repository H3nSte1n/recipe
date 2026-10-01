import { useEffect, useRef, useState } from 'react';
import { importRecipeFromPDF, importRecipeFromURL } from '../services/recipeService';
import { Recipe } from '../types/recipe';
import '../styles/ImportRecipeModal.css';

interface ImportRecipeModalProps {
  onClose: () => void;
  onManual: () => void;
  onImported: (recipe: Recipe) => void;
}

type ImportMethod = 'url' | 'pdf' | null;
const maxPDFBytes = 20 * 1024 * 1024;

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    if (error.message.includes('403')) return 'Verify your email before importing recipes.';
    if (error.message.includes('413')) return 'This PDF is too large. Choose a file under 20 MB.';
    if (error.message.includes('400')) return 'The link or PDF could not be read. Check it and try again.';
  }
  return 'Could not import this recipe. Check the link or file and try again.';
}

export default function ImportRecipeModal({ onClose, onManual, onImported }: ImportRecipeModalProps) {
  const [method, setMethod] = useState<ImportMethod>(null);
  const [url, setURL] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const controller = useRef<AbortController | null>(null);

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      controller.current?.abort();
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        controller.current?.abort();
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  function close() {
    controller.current?.abort();
    onClose();
  }

  async function handleImport(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (isImporting || !method) return;

    if (method === 'url') {
      try {
        const parsedURL = new URL(url.trim());
        if (!['http:', 'https:'].includes(parsedURL.protocol) || !parsedURL.hostname) throw new Error();
      } catch {
        setError('Enter a valid http or https recipe link.');
        return;
      }
    } else if (!file || (file.type !== 'application/pdf' && !file.name.toLowerCase().endsWith('.pdf'))) {
      setError('Choose a PDF file.');
      return;
    } else if (file.size > maxPDFBytes) {
      setError('This PDF is too large. Choose a file under 20 MB.');
      return;
    }

    setError(null);
    setIsImporting(true);
    const request = new AbortController();
    controller.current = request;
    try {
      const recipe = method === 'url'
        ? await importRecipeFromURL(url.trim(), request.signal)
        : await importRecipeFromPDF(file!, request.signal);
      onImported(recipe);
    } catch (importError) {
      if (!request.signal.aborted) setError(errorMessage(importError));
    } finally {
      controller.current = null;
      setIsImporting(false);
    }
  }

  return (
    <div className="import-recipe-modal" onClick={close}>
      <section className="import-recipe-modal__card" role="dialog" aria-modal="true" aria-labelledby="import-recipe-title" onClick={(event) => event.stopPropagation()}>
        <button className="import-recipe-modal__close" type="button" aria-label="Close" onClick={close}>×</button>
        <span className="import-recipe-modal__eyebrow">YOUR KITCHEN</span>
        <h2 className="import-recipe-modal__title" id="import-recipe-title">Add a recipe</h2>
        <p className="import-recipe-modal__intro">Start from scratch or bring in a recipe to review and publish.</p>

        <div className="import-recipe-modal__methods" aria-label="Choose how to add a recipe">
          <button className="import-recipe-modal__method" type="button" onClick={onManual} disabled={isImporting}>
            <span className="import-recipe-modal__method-name">Create manually</span>
            <span className="import-recipe-modal__method-arrow" aria-hidden="true">↗</span>
          </button>
          <button className={`import-recipe-modal__method${method === 'url' ? ' import-recipe-modal__method--selected' : ''}`} type="button" aria-pressed={method === 'url'} disabled={isImporting} onClick={() => { setMethod('url'); setError(null); }}>
            <span className="import-recipe-modal__method-name">Import a link</span>
            <span className="import-recipe-modal__method-arrow" aria-hidden="true">↗</span>
          </button>
          <button className={`import-recipe-modal__method${method === 'pdf' ? ' import-recipe-modal__method--selected' : ''}`} type="button" aria-pressed={method === 'pdf'} disabled={isImporting} onClick={() => { setMethod('pdf'); setError(null); }}>
            <span className="import-recipe-modal__method-name">Import a PDF</span>
            <span className="import-recipe-modal__method-arrow" aria-hidden="true">↗</span>
          </button>
        </div>

        {method && (
          <form className="import-recipe-modal__form" onSubmit={(event) => void handleImport(event)}>
            {method === 'url' ? (
              <label className="import-recipe-modal__label" htmlFor="import-recipe-url">
                Recipe link
                <input id="import-recipe-url" className="import-recipe-modal__input" type="url" inputMode="url" placeholder="https://example.com/recipe" value={url} onChange={(event) => { setURL(event.target.value); setError(null); }} disabled={isImporting} required autoFocus />
              </label>
            ) : (
              <label className="import-recipe-modal__label" htmlFor="import-recipe-file">
                PDF file
                <input id="import-recipe-file" className="import-recipe-modal__file" type="file" accept=".pdf,application/pdf" onChange={(event) => { setFile(event.target.files?.[0] ?? null); setError(null); }} disabled={isImporting} required />
                <span className="import-recipe-modal__hint">PDF with selectable text · 20 MB maximum</span>
              </label>
            )}
            {error && <p className="import-recipe-modal__error" role="alert">{error}</p>}
            <button className="import-recipe-modal__submit" type="submit" disabled={isImporting || (method === 'url' ? !url.trim() : !file)}>
              {isImporting ? 'Importing…' : 'Import recipe'}
            </button>
            <p className="import-recipe-modal__hint">You can edit everything before publishing.</p>
          </form>
        )}
      </section>
    </div>
  );
}
