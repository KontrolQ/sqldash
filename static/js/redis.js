(() => {
  'use strict';
  const root = document.querySelector('[data-redis-page]');
  if (!root) return;
  const query = selector => root.querySelector(selector);
  const api = root.dataset.redisApi;
  let leaving = false;
  window.addEventListener('pageshow',()=>{leaving=false;});
  const navigation = document.querySelector('[data-redis-navigation]');
  const shell = root.closest('.shell');
  const closeNavigation = () => {
    shell?.classList.remove('redis-navigation-open');
    navigation?.setAttribute('aria-expanded','false');
    query('[data-redis-navigation-close]')?.setAttribute('hidden','');
  };
  navigation?.addEventListener('click', () => {
    const open = shell?.classList.toggle('redis-navigation-open');
    navigation.setAttribute('aria-expanded',String(!!open));
    query('[data-redis-navigation-close]')?.toggleAttribute('hidden',!open);
  });
  query('[data-redis-navigation-close]')?.addEventListener('click',closeNavigation);
  document.addEventListener('keydown',event=>{if(event.key==='Escape')closeNavigation();});
  const encoder = new TextEncoder();
  const decoder = new TextDecoder('utf-8', {fatal: true, ignoreBOM: true});
  const element = (tag, text, className) => {
    const node = document.createElement(tag);
    if (text !== undefined && text !== null) node.textContent = String(text);
    if (className) node.className = className;
    return node;
  };
  function error(message) {
    const target = query('[data-redis-error]');
    (target.querySelector('[data-redis-error-message]') || target).textContent = message || '';
    target.hidden = !message;
  }
  function responseProblem(response,result) {
    return result?.error || result?.message || ({401:'Your session expired. Sign in again before retrying.',403:'Your account no longer has permission for this Redis action.',404:'This Redis database or action is no longer available.',409:'The Redis instance changed or is not ready. Inspect its current state before retrying.',503:'Redis is currently unavailable. Your page draft is retained.'})[response.status] || 'Redis request failed ('+response.status+').';
  }
  async function request(path, body, options = {}) {
    const response = await fetch(api + path, {
      method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', cache: 'no-store',
      headers: {Accept: 'application/json', ...(body === undefined ? {} : {'Content-Type': 'application/json'})},
      ...(body === undefined ? {} : {body: JSON.stringify(body)}), ...options
    });
    if (!response.headers.get('Content-Type')?.includes('application/json')) throw new Error(response.ok ? 'The server did not return a Redis response. Check your session and reconnect.' : responseProblem(response));
    const result = await response.json();
    if (!response.ok) throw new Error(responseProblem(response,result));
    return result;
  }
  const safely = operation => async event => {
    if (event?.preventDefault) event.preventDefault();
    error('');
    try { await operation(event); } catch (problem) { if (problem.name !== 'AbortError') error(problem.message || 'Redis request failed.'); }
  };
  function encodeBytes(bytes) {
    let binary = '';
    for (let offset = 0; offset < bytes.length; offset += 8192) binary += String.fromCharCode(...bytes.subarray(offset, offset + 8192));
    return {encoding: 'base64', data: btoa(binary)};
  }
  function decodeBytes(value) {
    if (!value || value.encoding !== 'base64') throw new Error('Unsupported binary encoding from Redis.');
    const binary = atob(value.data);
    return Uint8Array.from(binary, character => character.charCodeAt(0));
  }
  function textBytes(value, format = 'text') {
    if (format === 'text' || format === 'json') {
      if (/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(^|[^\uD800-\uDBFF])[\uDC00-\uDFFF]/u.test(value)) throw new Error('Text contains an unpaired Unicode surrogate. Use hex or base64 for those bytes.');
      if (format === 'json') JSON.parse(value);
      return encodeBytes(encoder.encode(value));
    }
    if (format === 'hex') {
      const compact = value.replace(/\s/g, '');
      if (compact.length % 2 || /[^a-f0-9]/i.test(compact)) throw new Error('Hex must contain complete pairs of hexadecimal digits.');
      return encodeBytes(Uint8Array.from(compact.match(/../g) || [], pair => parseInt(pair, 16)));
    }
    if (format === 'base64') {
      const compact = value.replace(/\s/g, '');
      if (!/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(compact)) throw new Error('Enter canonical padded base64.');
      const decoded = decodeBytes({encoding: 'base64', data: compact});
      if (encodeBytes(decoded).data !== compact) throw new Error('Enter canonical padded base64.');
      return {encoding: 'base64', data: compact};
    }
    throw new Error('Unknown byte encoding.');
  }
  function displayBytes(value, format = 'text') {
    if (format === 'base64') return value.data;
    const bytes = decodeBytes(value);
    if (format === 'hex') return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('');
    return decoder.decode(bytes);
  }
  function label(value) {
    try {
      const text = displayBytes(value);
      return text.replace(/[\x00-\x1f\x7f-\x9f‪-‮⁦-⁩]/g, character => '\\u' + character.charCodeAt(0).toString(16).padStart(4, '0')) || '(empty key)';
    } catch { return 'base64:' + value.data; }
  }
  function prettyJSON(source) {
    if(source.length > 262144) throw new Error('JSON is too large for the formatted view.');
    JSON.parse(source);
    let output = '', indentation = 0, quoted = false, escaped = false;
    const newline = () => { output = output.trimEnd() + '\n' + '  '.repeat(indentation); };
    for (let index = 0; index < source.length; index++) {
      if(output.length > 1048576) throw new Error('JSON exceeds the formatted view size.');
      const character = source[index];
      if (quoted) {
        output += character;
        if (escaped) escaped = false;
        else if (character === '\\') escaped = true;
        else if (character === '"') quoted = false;
        continue;
      }
      if (/\s/.test(character)) continue;
      if (character === '"') { quoted = true; output += character; }
      else if (character === '{' || character === '[') {
        output += character;
        const next = source.slice(index + 1).trimStart()[0];
        if (next !== '}' && next !== ']') { indentation++; if(indentation > 64 || output.length + indentation * 2 > 1048576) throw new Error('JSON exceeds the formatted view depth or size.'); newline(); }
      } else if (character === '}' || character === ']') {
        const previous = output.trimEnd().slice(-1);
        if (previous !== '{' && previous !== '[') { indentation = Math.max(0, indentation - 1); newline(); }
        output += character;
      } else if (character === ',') { output += character; newline(); }
      else if (character === ':') output += ': ';
      else output += character;
    }
    return output;
  }
  function table(headers, rows) {
    const output = element('table', null, 'table');
    const head = element('thead'), headRow = element('tr');
    for (const header of headers) { const cell = element('th', header); cell.scope = 'col'; headRow.append(cell); }
    head.append(headRow); output.append(head);
    const body = element('tbody');
    for (const row of rows) {
      const rowNode = element('tr');
      for (const value of row) { const cell = element('td'); if (value instanceof Node) cell.append(value); else cell.textContent = value ?? '—'; rowNode.append(cell); }
      body.append(rowNode);
    }
    if (!rows.length) { const row = element('tr'), cell = element('td', 'No entries to display.'); cell.colSpan = headers.length; row.append(cell); body.append(row); }
    output.append(body); return output;
  }
  function enhancedSelect(native) {
    const field=element('div',null,'select is-wide');field.dataset.select='';
    const trigger=element('button',null,'select-trigger is-wide'+(native.classList.contains('is-small')?' is-small':''));trigger.type='button';trigger.dataset.selectTrigger='';trigger.setAttribute('aria-haspopup','listbox');trigger.setAttribute('aria-expanded','false');
    const label=element('span');label.dataset.selectLabel='';trigger.append(label);
    const arrow=document.createElementNS('http://www.w3.org/2000/svg','svg');arrow.setAttribute('viewBox','0 0 24 24');const path=document.createElementNS('http://www.w3.org/2000/svg','path');path.setAttribute('d','m6 9 6 6 6-6');arrow.append(path);trigger.append(arrow);
    const content=element('div',null,'select-content');content.dataset.selectContent='';content.setAttribute('role','listbox');content.hidden=true;
    field.append(native,trigger,content);return field;
  }
  for (const native of root.querySelectorAll('select.input')) {
    const parent = native.parentNode, sibling = native.nextSibling;
    native.classList.remove('input');
    parent.insertBefore(enhancedSelect(native), sibling);
  }
  document.dispatchEvent(new Event('sqldash:scan'));
  const dialogNode = query('[data-redis-dialog]');
  function dialog({title, description, fields = [], submit = 'Apply', onSubmit, secret = false, destructive = false}) {
    if (dialogNode.open) dialogNode.close();
    query('#redis-dialog-title').textContent = title;
    const body = query('[data-redis-dialog-body]'); body.replaceChildren();
    if (description) body.append(element('p', description, 'field-note'));
    const controls = {};
    for (const field of fields) {
      const wrapper = element('div', null, 'field');
      const fieldLabel = element('label', field.label, 'label'); fieldLabel.htmlFor='redis-dialog-'+field.name; wrapper.append(fieldLabel);
      const control = element(field.type === 'textarea' ? 'textarea' : field.options ? 'select' : 'input');
      control.id='redis-dialog-'+field.name;
      control.className = field.options ? '' : field.type === 'textarea' ? 'input mono' : 'input';
      if (field.options) for (const option of field.options) { const entry = element('option', typeof option === 'string' ? option : option.label); entry.value = typeof option === 'string' ? option : option.value; control.append(entry); }
      else if (field.type !== 'textarea') control.type = field.type || 'text';
      control.name = field.name;
      if (field.value !== undefined) control.value = field.value;
      control.required = !!field.required;
      control.autocomplete = 'off'; control.spellcheck = false;
      if (field.readOnly) control.readOnly = true;
      if (field.placeholder) control.placeholder = field.placeholder;
      wrapper.append(field.options ? enhancedSelect(control) : control);
      if (field.type === 'password') {
        const actions = element('span', null, 'redis-actions');
        const reveal = element('button', 'Reveal', 'button is-outline is-small'); reveal.type = 'button'; reveal.setAttribute('aria-label', 'Reveal ' + field.label); reveal.setAttribute('aria-pressed', 'false');
        reveal.addEventListener('click', () => { const visible = control.type === 'password'; control.type = visible ? 'text' : 'password'; reveal.textContent = visible ? 'Hide' : 'Reveal'; reveal.setAttribute('aria-label', (visible ? 'Hide ' : 'Reveal ') + field.label); reveal.setAttribute('aria-pressed', String(visible)); });
        const copyButton = element('button', 'Copy', 'button is-outline is-small'); copyButton.type = 'button'; copyButton.setAttribute('aria-label', 'Copy ' + field.label);
        copyButton.addEventListener('click', async () => { try { await copy(control.value); copyButton.textContent = 'Copied'; } catch { control.focus(); control.select(); copyButton.textContent = 'Select and copy'; } });
        actions.append(reveal, copyButton); wrapper.append(actions);
      }
      body.append(wrapper); controls[field.name] = control;
    }
    const submitButton = query('[data-redis-dialog-submit]'); submitButton.textContent = submit; submitButton.classList.toggle('is-destructive',destructive);
    const dialogError = query('[data-redis-dialog-error]'), dialogMessage = dialogError.querySelector('[data-redis-dialog-error-message]') || dialogError; dialogError.hidden = true; dialogMessage.textContent = '';
    return new Promise(resolve => {
      let settled = false, submitting = false;
      const cancelButton=query('[data-redis-dialog-cancel]');
      dialogNode.oncancel = event => { if(submitting)event.preventDefault(); };
      dialogNode.onclose = () => {
        for (const control of Object.values(controls)) control.value = '';
        body.replaceChildren(); query('[data-redis-dialog-form]').onsubmit = null;
        resolve(settled);
      };
      cancelButton.onclick = () => { if(!submitting)dialogNode.close(); };
      cancelButton.hidden = secret; cancelButton.disabled = false;
      query('[data-redis-dialog-form]').onsubmit = async event => {
        event.preventDefault(); if (submitButton.disabled) return;
        submitButton.disabled = true; submitting = true; cancelButton.disabled = true;
        try { await onSubmit?.(Object.fromEntries(Object.entries(controls).map(([name, control]) => [name, control.value])), controls); settled = true; dialogNode.close(); }
        catch (problem) { dialogMessage.textContent = problem.message; dialogError.hidden = false; }
        finally { submitButton.disabled = false; submitting = false; cancelButton.disabled = false; }
      };
      submitButton.disabled = false; dialogNode.showModal(); document.dispatchEvent(new Event('sqldash:scan'));
    });
  }
  const confirm = (title, description, onSubmit, submit = 'Confirm') => dialog({title, description, onSubmit, submit, destructive:/^(Delete|Remove|Pop|Reset)/.test(submit)});
  const copy = async value => { if (!navigator.clipboard) throw new Error('Clipboard access is unavailable. Select and copy the displayed value.'); await navigator.clipboard.writeText(value); };
  window.SqlDashRedis = {root, query, request, safely, error, element, encodeBytes, decodeBytes, textBytes, displayBytes, label, prettyJSON, table, dialog, confirm, copy};
  window.addEventListener('pagehide', () => { leaving=true; if (dialogNode.open) dialogNode.close(); });

  const page = root.dataset.redisPage;

  function definition(target, entries, className) {
    target.replaceChildren();
    for (const [name, value] of entries) { const group = element('div', null, className), term = element('dt', name), detail = element('dd', value ?? '—'); if (className === 'tile') { term.className = 'tile-label'; detail.className = 'tile-value'; } group.append(term, detail); target.append(group); }
  }
  const timestamp = value => new Date(value).toLocaleString();
  const decimal = value => value === null || value === undefined ? '—' : Number(value).toLocaleString(undefined, {maximumFractionDigits: 2});
  function chart(target, samples, field) {
    const memory = field === 'used_memory';
    const points = [];
    let previousRun = null;
    for (const sample of samples) {
      const label = new Date(sample.recordedAt).toLocaleTimeString();
      if (previousRun !== null && (sample.runId !== previousRun || sample.counterReset)) points.push({label, values: [null], note: 'Server restart or counter reset'});
      const raw = memory ? sample.fields.used_memory : sample[field];
      const value = raw === null || raw === undefined ? null : Number(raw);
      points.push({label, values: [Number.isFinite(value) ? value : null]});
      previousRun = sample.runId;
    }
    if (points.filter(point => point.values[0] !== null).length < 2) {
      if (!target.querySelector('[data-chart]')) target.replaceChildren(element('p', 'Collecting measurements…', 'field-note'));
      return;
    }
    let holder = target.querySelector('[data-chart]');
    if (!holder) {
      holder = element('div', null, 'chart'); holder.dataset.chart = '';
      const key = element('div', null, 'chart-key'); key.dataset.chartKey = '';
      const plot = element('div', null, 'chart-plot');
      const canvas = element('div'); canvas.dataset.chartCanvas = '';
      const tip = element('div', null, 'chart-tip'); tip.dataset.chartTip = ''; tip.hidden = true;
      plot.append(canvas, tip); holder.append(key, plot); target.replaceChildren(holder);
    }
    window.SqlDashCharts.render(holder, {kind: 'line', series: [memory ? 'Memory used' : 'Commands / second'], unit: memory ? 'bytes' : '', table: true, points});
  }
  if (page === 'insights') {
    let pending = false;
    const refresh = async () => {
      if (pending || document.hidden) return; pending = true;
      try {
        const result = await request('/insights'), sample = result.current, fields = sample.fields;
        query('[data-redis-measured-at]').textContent = 'Measured ' + timestamp(sample.recordedAt) + ' · run ' + sample.runId;
        const note = query('[data-redis-history-note]'); note.textContent = result.historyProblem || (sample.counterReset ? 'Server identity or counters changed. Rates restart from the next comparable measurement.' : ''); note.hidden = !note.textContent;
        definition(query('[data-redis-metrics]'), [['Commands / second', decimal(sample.commandsPerSecond)], ['Interval cache hit rate', sample.hitRate == null ? '—' : decimal(sample.hitRate) + '%'], ['Memory used', window.SqlDashCharts.format(Number(fields.used_memory), 'bytes')], ['Evicted keys · total', fields.evicted_keys]], 'tile');
        const history = [...result.history]; if (history.at(-1)?.recordedAt !== sample.recordedAt) history.push(sample);
        for (const target of root.querySelectorAll('[data-redis-chart]')) chart(target, history, target.dataset.redisChart);
        query('[data-redis-history]').replaceChildren(table(['Measured', 'Commands / s', 'Hit rate %', 'Memory bytes', 'Input bytes / s', 'Output bytes / s', 'Run / reset'], history.map(entry => [timestamp(entry.recordedAt), decimal(entry.commandsPerSecond), decimal(entry.hitRate), entry.fields.used_memory, decimal(entry.inputBytesPerSecond), decimal(entry.outputBytesPerSecond), entry.runId + (entry.counterReset ? ' · reset' : '')])));
        const labels = {redis_version:'Version', uptime_in_seconds:'Uptime · seconds', connected_clients:'Connected clients', blocked_clients:'Blocked clients', maxmemory:'Memory limit · bytes', used_memory_rss:'Resident memory · bytes', mem_fragmentation_ratio:'Fragmentation ratio', db0:'Keyspace', expired_keys:'Expired keys · total', rejected_connections:'Rejected connections · total', role:'Replication role', master_link_status:'Primary link', loading:'Loading snapshot', rdb_last_save_time:'Last snapshot · Unix seconds', rdb_last_bgsave_status:'Last snapshot result', rdb_bgsave_in_progress:'Snapshot in progress', aof_enabled:'AOF enabled', aof_last_write_status:'AOF write result', aof_last_bgrewrite_status:'AOF rewrite result', aof_rewrite_in_progress:'AOF rewrite in progress'};
        definition(query('[data-redis-server]'), Object.entries(labels).filter(([field]) => fields[field] !== undefined).map(([field, name]) => [name, fields[field]]));
        error('');
      } finally { pending = false; }
    };
    query('[data-redis-insights-refresh]').addEventListener('click', safely(refresh));
    let timer;
    const poll = () => {clearInterval(timer);timer=setInterval(()=>{if(query('[data-redis-insights-live]').checked)safely(refresh)();},5000);};
    window.addEventListener('pagehide',()=>clearInterval(timer));
    window.addEventListener('pageshow',event=>{if(event.persisted){poll();safely(refresh)();}});
    poll();safely(refresh)();
  }
  if (page === 'queries') {
    const refresh = safely(async () => {
      const result = await request('/commands');
      query('[data-redis-measured-at]').textContent = 'Measured ' + timestamp(result.recordedAt) + ' · run ' + result.runId;
      query('[data-redis-commands]').replaceChildren(table(['Command','Calls','CPU · µs','CPU / call · µs','Rejected','Failed'], result.commands.map(command => [command.name,command.calls,command.microseconds,command.microsecondsPerCall,command.rejectedCalls,command.failedCalls])));
      if(result.truncated) query('[data-redis-commands]').append(element('p','Bounded command view: only 2,048 command counter entries are included.','redis-note'));
    }); query('[data-redis-commands-refresh]').addEventListener('click', refresh); refresh();
  }
  if (page === 'slowlog') {
    const refresh = async () => {
      const result = await request('/slow-log');
      query('[data-redis-slowlog]').replaceChildren(table(['Time','Duration · µs','Command arguments','Client','ID'], result.entries.map(entry => [timestamp(entry.time),entry.microseconds,element('pre',entry.arguments.map(value => label(value)).join(' ') + (entry.truncated ? '\n[arguments truncated]' : '')),entry.name || entry.client,entry.id])));
      if (result.truncated) query('[data-redis-slowlog]').append(element('p','Showing a bounded recent window. Older entries may not be included.','redis-note'));
    };
    query('[data-redis-slowlog-refresh]').addEventListener('click', safely(refresh));
    query('[data-redis-slowlog-reset]')?.addEventListener('click', safely(() => confirm('Reset slow log?', 'Remove every slow-log entry for this database. This does not delete Redis data.', async () => { await request('/slow-log/reset', {}); await refresh(); }, 'Reset slow log')));
    safely(refresh)();
  }

  if (page === 'monitor' || page === 'pubsub') {
    let controller = null, events = [], bufferBytes = 0, removed = 0n, serverDropped = 0n, pendingRender = false;
    const output = query('[data-redis-stream-output]');
    const state = value => { const target = query('[data-redis-stream-state]'); if (target && target.textContent !== value) target.textContent = value; };
    const render = () => {
      pendingRender = false; if (!output) return;
      output.replaceChildren();
      const format = query('[data-redis-message-format]')?.value || 'text';
      if (page === 'pubsub') {
        const rows = events.map(event => {
          let payload;
          try { payload = displayBytes(event.payload, format); if (format === 'text') { try { payload = prettyJSON(payload); } catch {} } }
          catch { payload = '[binary · base64] ' + event.payload.data; }
          const copyMessage = element('button', 'Copy', 'button is-ghost is-small'); copyMessage.type = 'button';
          copyMessage.addEventListener('click', safely(() => copy(payload)));
          return [timestamp(event.recordedAt), label(event.channel), element('pre', payload + (event.truncated ? '\n[truncated]' : '')), copyMessage];
        });
        output.append(table(['Time', 'Channel', 'Message', 'Action'], rows));
      }
      for (const event of page === 'pubsub' ? [] : events) {
        const entry = element('div', null, 'redis-live-entry');
        entry.append(element('pre', event.line));
        if (event.truncated) entry.append(element('span','[truncated]','badge is-outline'));
        output.append(entry);
      }
      if (!events.length && page !== 'pubsub') output.append(element('p','No live messages in the buffer.','redis-note'));
      query('[data-redis-stream-count]').textContent = events.length + ' / 200 buffered · ' + Math.ceil(bufferBytes/1024) + ' / 2048 KiB · ' + removed + ' evicted locally · ' + serverDropped + ' dropped upstream';
      output.scrollTop = output.scrollHeight;
    };
    const stop = () => { controller?.abort(); controller = null; state('Stopped'); query('[data-redis-stream-stop]')?.setAttribute('disabled',''); query('[data-redis-monitor-start]')?.removeAttribute('disabled'); query('[data-redis-subscribe-form] button[type="submit"]')?.removeAttribute('disabled'); };
    const start = async (path, body) => {
      stop(); const running = new AbortController(); controller = running;
      state('Connecting…'); query('[data-redis-stream-stop]')?.removeAttribute('disabled'); query('[data-redis-monitor-start]')?.setAttribute('disabled',''); query('[data-redis-subscribe-form] button[type="submit"]')?.setAttribute('disabled','');
      try {
        const response = await fetch(api + path,{method:'POST',credentials:'same-origin',cache:'no-store',headers:{'Content-Type':'application/json',Accept:'text/event-stream'},body:JSON.stringify(body),signal:running.signal});
        if (!response.ok || !response.headers.get('Content-Type')?.includes('text/event-stream')) { let problem; try { problem = await response.json(); } catch {} throw new Error(responseProblem(response,problem)); }
        const reader = response.body.getReader(), textDecoder = new TextDecoder(); let buffer = '';
        try {
          while (true) {
            const chunk = await reader.read(); if (chunk.done) break;
            buffer += textDecoder.decode(chunk.value,{stream:true});
            if (buffer.length > 2*1024*1024) throw new Error('Live event exceeded the bounded stream size.');
            let boundary;
            while ((boundary = buffer.indexOf('\n\n')) >= 0) {
              const block = buffer.slice(0,boundary); buffer = buffer.slice(boundary+2);
              let kind = 'message', data = '';
              for (const line of block.split('\n')) { if (line.startsWith('event:')) kind = line.slice(6).trim(); else if (line.startsWith('data:')) data += line.slice(5).trimStart(); }
              if (kind === 'ready') { state('Live stream open · waiting for Redis events'); continue; }
              if (!data) continue;
              const event = JSON.parse(data);
              if (kind === 'stopped') { state(event.error); return; }
              if (event.dropped) { try { serverDropped += BigInt(event.dropped); } catch {} }
              state('Connected · live only');
              event.bufferSize = encoder.encode(data).byteLength; events.push(event); bufferBytes += event.bufferSize;
              while (events.length > 200 || bufferBytes > 2*1024*1024) { bufferBytes -= events.shift().bufferSize; removed++; }
              if (!pendingRender) { pendingRender = true; requestAnimationFrame(render); }
            }
          }
          state('Disconnected · start again to reconnect');
        } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
      } finally { if (controller === running) { controller = null; query('[data-redis-stream-stop]')?.setAttribute('disabled',''); query('[data-redis-monitor-start]')?.removeAttribute('disabled'); query('[data-redis-subscribe-form] button[type="submit"]')?.removeAttribute('disabled'); } }
    };
    query('[data-redis-monitor-start]')?.addEventListener('click', safely(() => confirm('Start command monitor?', 'MONITOR increases server work and may expose private values. Authentication and secret control commands are filtered, but application data remains visible.', () => { safely(() => start('/monitor',{}))(); }, 'Start monitor')));
    query('[data-redis-stream-stop]')?.addEventListener('click', stop);
    query('[data-redis-stream-clear]')?.addEventListener('click', () => { events=[];bufferBytes=0;removed=0n;serverDropped=0n;render(); });
    query('[data-redis-message-format]')?.addEventListener('change',render);
    query('[data-redis-subscribe-form]')?.addEventListener('submit',safely(event => { const form=event.currentTarget; return start('/pubsub/subscribe',{channel:textBytes(form.elements.channel.value,form.elements.encoding.value),pattern:form.elements.pattern.checked}); }));
    query('[data-redis-publish-form]')?.addEventListener('submit',safely(async event => { const form=event.currentTarget; const result=await request('/pubsub/publish',{channel:textBytes(form.elements.channel.value,form.elements.channelEncoding.value),payload:textBytes(form.elements.payload.value,form.elements.encoding.value)}); query('[data-redis-publish-result]').textContent='Delivered to '+result.receivers+' active subscription(s).'; }));
    window.addEventListener('pagehide',() => {stop();events=[];bufferBytes=0;removed=0n;serverDropped=0n;if(output)render();});
  }

  if (page === 'connections') {
    const refresh = async () => {
      if (!query('[data-redis-clients]')) return;
      const result=await request('/clients');
      const manage = root.dataset.redisActivityManage === 'true';
      const headers = ['ID','Client','User','Address','Age · s','Idle · s','Last command','Flags'];
      if (manage) headers.push('Action');
      const rows = result.clients.map(client => {
        const row = [client.id,client.name,client.username,client.address,client.age,client.idle,client.command,client.flags];
        if (manage) {
          const disconnect = element('button','Disconnect','button is-outline is-small'); disconnect.type = 'button';
          disconnect.addEventListener('click',safely(async () => { await confirm('Disconnect Redis client?','This closes client '+client.id+' only. Its credential remains valid for reconnection.',async () => { await request('/clients/terminate',{id:client.id,runId:result.runId}); await refresh(); },'Disconnect'); }));
          row.push(disconnect);
        }
        return row;
      });
      query('[data-redis-clients]').replaceChildren(table(headers,rows));
      if(result.truncated) query('[data-redis-clients]').append(element('p','Client list truncated at 1,000 entries.','redis-note'));
    };
    query('[data-redis-clients-refresh]')?.addEventListener('click',safely(refresh)); safely(refresh)();
    for(const form of root.querySelectorAll('[data-redis-credential-form]')) form.addEventListener('submit',safely(async()=>{
      if(form.dataset.redisPending==='true')return;
      form.dataset.redisPending='true';
      const submitters=[...form.querySelectorAll('button[type="submit"]')];
      for(const button of submitters)button.disabled=true;
      try {
      const perform=async()=>{
        const body=new FormData(form), expiry=body.get('expiresAt');
        if(expiry) body.set('expiresAt',new Date(expiry).toISOString());
        const response=await fetch(form.action,{method:'POST',credentials:'same-origin',cache:'no-store',headers:{Accept:'application/json'},body});
        if(!response.headers.get('Content-Type')?.includes('application/json')) throw new Error(response.ok ? 'Credential action did not return a JSON result. Check your session.' : responseProblem(response));
        const result=await response.json(); if(!response.ok) throw new Error(responseProblem(response,result));
        if(result.password) {
          await dialog({title:'Native Redis credential',description:'Copy this password now. It is not stored in this page or in browser storage. Closing this dialog clears it.',submit:'Close',secret:true,fields:[{name:'username',label:'Username',value:result.username,readOnly:true},{name:'password',label:'Password',type:'password',value:result.password,readOnly:true},{name:'url',label:'Connection URL',type:'password',value:result.url,readOnly:true}],onSubmit:()=>{}});
          result.password='';result.url='';
        }
        if(!leaving)window.location.reload();
      };
      if(form.dataset.redisCredentialConfirm && !await confirm('Change native credential?',form.dataset.redisCredentialConfirm,()=>{})) return;
      await perform();
      } finally {delete form.dataset.redisPending;for(const button of submitters)button.disabled=false;}
    }));
  }
})();
