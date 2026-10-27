(() => {
  'use strict';
  const terminal = document.querySelector('[data-redis-terminal]');
  if (!terminal) return;
  const input = terminal.querySelector('[data-redis-terminal-input]');
  const output = terminal.querySelector('[data-redis-terminal-output]');
  const prompt = terminal.dataset.prompt;
  const status = document.querySelector('[data-redis-terminal-status]');
  const decoder = new TextDecoder('utf-8', {fatal: true});
  const history = [];
  let historyIndex = 0, savedDraft = '', pending = false, leaving = false;
  const maximumCommandLength = 65536;
  const clean = value => value.replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, character => '\\x' + character.charCodeAt(0).toString(16).padStart(2, '0'));

  function node(tag, text, className) {
    const result = document.createElement(tag);
    result.textContent = text;
    if (className) result.className = className;
    return result;
  }

  function readInput() {
    return input.innerText.replace(/\r\n?/g, '\n').replace(/ /g, ' ');
  }

  function focusInput(atEnd = false) {
    input.focus({preventScroll: true});
    if (atEnd) {
      const range = document.createRange();
      range.selectNodeContents(input);
      range.collapse(false);
      const selection = window.getSelection();
      selection.removeAllRanges();
      selection.addRange(range);
    }
    input.scrollIntoView({block: 'nearest'});
  }

  function replaceInput(value) {
    input.textContent = value;
    focusInput(true);
  }

  function insertText(value) {
    input.focus();
    const selection = window.getSelection();
    if (!selection.rangeCount || !input.contains(selection.anchorNode)) focusInput(true);
    const range = window.getSelection().getRangeAt(0);
    range.deleteContents();
    const text = document.createTextNode(value);
    range.insertNode(text);
    range.setStartAfter(text);
    range.collapse(true);
    selection.removeAllRanges();
    selection.addRange(range);
    input.dispatchEvent(new Event('input', {bubbles: true}));
  }

  function clearTranscript() {
    output.replaceChildren();
  }

  function entry(command) {
    const result = node('div', '', 'redis-console-entry');
    const line = node('div', '', 'redis-console-line');
    line.append(node('span', prompt, 'redis-console-prompt'), node('span', clean(command), 'redis-console-command'));
    result.append(line);
    output.append(result);
    while (output.children.length > 100) output.firstElementChild.remove();
    return result;
  }

  function print(target, message, tone = '') {
    target.append(node('pre', clean(message), 'redis-console-reply' + (tone ? ' is-' + tone : '')));
  }

  function formatReply(reply, depth = 0) {
    const indentation = '  '.repeat(depth);
    if (reply.type === 'bytes') {
      const bytes = Uint8Array.from(atob(reply.bytes.data), character => character.charCodeAt(0));
      let text;
      try { text = decoder.decode(bytes); }
      catch { text = '(binary) ' + Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join(' '); }
      return indentation + (text || '""') + (reply.truncated ? ' [truncated]' : '');
    }
    if (reply.type === 'array') {
      if (!reply.items?.length) return indentation + '(empty array)';
      return reply.items.map((item, index) => indentation + (index + 1) + ') ' + formatReply(item, depth + 1).trimStart()).join('\n') + (reply.truncated ? '\n' + indentation + '[truncated]' : '');
    }
    if (reply.type === 'null') return indentation + '(nil)';
    if (reply.type === 'integer' || reply.type === 'double' || reply.type === 'boolean') return indentation + '(' + reply.type + ') ' + reply.text;
    if (reply.type === 'error') return indentation + '(error) ' + reply.text;
    return indentation + (reply.text ?? '(' + reply.type + ')');
  }

  async function execute() {
    if (pending) return;
    const command = readInput().trim();
    if (!command) { replaceInput(''); return; }
    if (command.toLowerCase() === 'clear') {
      clearTranscript();
      historyIndex = history.length;
      savedDraft = '';
      status.textContent = 'Ready';
      replaceInput('');
      return;
    }
    const resultEntry = entry(command);
    input.textContent = '';
    if (command.length > maximumCommandLength) {
      print(resultEntry, 'Command input exceeds 64 KiB. Nothing was sent.', 'error');
      focusInput(true);
      return;
    }
    if (history.at(-1) !== command) history.push(command);
    while (history.length > 100) history.shift();
    historyIndex = history.length;
    savedDraft = '';
    pending = true;
    input.contentEditable = 'false';
    terminal.setAttribute('aria-busy', 'true');
    status.textContent = 'Running…';
    try {
      const response = await fetch(terminal.dataset.api + '/console', {
        method: 'POST', credentials: 'same-origin', cache: 'no-store',
        headers: {'Content-Type': 'application/json', Accept: 'application/json'},
        body: JSON.stringify({commands: command}), signal: AbortSignal.timeout(25000)
      });
      if (!response.headers.get('Content-Type')?.includes('application/json')) throw new Error('The console did not receive a Redis response. Check your session before retrying.');
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || result.message || 'Redis request failed (' + response.status + ').');
      for (const reply of result.replies) print(resultEntry, formatReply(reply), reply.type === 'error' ? 'error' : '');
      if (result.truncated) print(resultEntry, 'Output truncated; remaining commands were not sent.', 'muted');
      if (result.discarded) print(resultEntry, 'Incomplete transaction discarded.', 'muted');
      status.textContent = result.milliseconds + ' ms';
    } catch (problem) {
      print(resultEntry, problem.name === 'TimeoutError' ? 'Request timed out. A command may have executed; inspect its result before retrying.' : problem.message || 'Redis request failed.', 'error');
      status.textContent = 'Request failed';
    } finally {
      pending = false;
      input.contentEditable = 'plaintext-only';
      terminal.removeAttribute('aria-busy');
      if (!leaving) focusInput(true);
    }
  }

  input.addEventListener('keydown', event => {
    if (event.isComposing) return;
    const modified = event.ctrlKey || event.metaKey;
    if (event.key === 'Enter') {
      event.preventDefault();
      if (event.shiftKey) insertText('\n');
      else execute();
      return;
    }
    if (modified && event.key.toLowerCase() === 'l') {
      event.preventDefault(); clearTranscript(); return;
    }
    if (modified && event.key.toLowerCase() === 'c' && !window.getSelection()?.toString()) {
      event.preventDefault();
      if (!pending) { entry(readInput() + '^C'); replaceInput(''); historyIndex = history.length; }
      return;
    }
    if (event.key === 'Escape') { event.preventDefault(); replaceInput(''); historyIndex = history.length; return; }
    if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return;
    const selection = window.getSelection();
    if (!selection?.isCollapsed || !input.contains(selection.anchorNode)) return;
    const range = document.createRange();
    range.selectNodeContents(input);
    range.setEnd(selection.anchorNode, selection.anchorOffset);
    const before = range.toString();
    const text = readInput();
    if (event.key === 'ArrowUp' && before.includes('\n') || event.key === 'ArrowDown' && text.slice(before.length).includes('\n')) return;
    event.preventDefault();
    if (historyIndex === history.length) savedDraft = text;
    historyIndex = Math.max(0, Math.min(history.length, historyIndex + (event.key === 'ArrowUp' ? -1 : 1)));
    replaceInput(historyIndex === history.length ? savedDraft : history[historyIndex]);
  });

  input.addEventListener('paste', event => {
    event.preventDefault();
    const text = event.clipboardData.getData('text/plain').replace(/\r\n?/g, '\n').replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, '');
    if (readInput().length + text.length > maximumCommandLength) { status.textContent = 'Paste exceeds 64 KiB'; return; }
    insertText(text);
  });
  input.addEventListener('input', () => {
    if (readInput().length > maximumCommandLength) replaceInput(readInput().slice(0, maximumCommandLength));
  });
  terminal.addEventListener('click', event => {
    if (!pending && !window.getSelection()?.toString() && !event.target.closest('a, button, [contenteditable]')) focusInput(true);
  });
  document.querySelector('[data-redis-terminal-clear]').addEventListener('click', () => { clearTranscript(); focusInput(true); });
  window.addEventListener('pagehide', () => { leaving = true; history.length = 0; savedDraft = ''; });
  window.addEventListener('pageshow', () => { leaving = false; });
  focusInput(true);
})();
