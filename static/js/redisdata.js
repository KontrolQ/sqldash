(() => {
  'use strict';
  const app = window.SqlDashRedis;
  if (!app || app.root.dataset.redisPage !== 'data') return;
  const {root,query,request:transport,safely,error,element,encodeBytes,decodeBytes,textBytes,displayBytes,label,prettyJSON,table,dialog,confirm,copy} = app;
  function hydrate(value) {
    for(const kind of ['hash','list','set','zset','stream']) if(value.type===kind && !value[kind]) value[kind]=[];
    return value;
  }
  const request = async (...argumentsList) => hydrate(await transport(...argumentsList));
  const write = root.dataset.redisWrite === 'true';
  const clone = value => structuredClone(value);
  const equalBytes = (first,second) => first?.data === second?.data;
  const bytes = value => ({encoding:'base64',data:value.data});
  const keyMap = new Map();
  const selectedEntries = new Set();
  const entryOrigins = new WeakMap();
  function entryOrigin(kind,entry) {
    if(entryOrigins.has(entry))return entryOrigins.get(entry);
    const source=kind==='hash'?current.hash.find(value=>equalBytes(value.field,entry.field)):kind==='zset'?current.zset.find(value=>equalBytes(value.member,entry.member)):kind==='set'?current.set.find(value=>equalBytes(value,entry)):null;
    const original=structuredClone(source||entry);entryOrigins.set(entry,original);return original;
  }
  let editingCell = null;
  const invalidInputs = new Map();
  function setInvalid(input, message) {
    if (message) invalidInputs.set(input, message); else invalidInputs.delete(input);
    invalid = invalidInputs.values().next().value || '';
    input.toggleAttribute('aria-invalid', !!message);
  }
  function clearInvalid() { invalidInputs.clear(); invalid = ''; }
  let scanCursor = '0', scanComplete = false, scanBound = false, lastDatabaseSize = '', activeScan = null, scanning = false, current = null, draft = null, ttlDraft = null;
  let view = 'pretty', encoding = 'text', invalid = '', pending = false, loading = false, displaySerial = 0;
  const editor = query('[data-redis-editor]');
  const scanForm = query('[data-redis-scan-form]');

  function payload(value) {
    if (!value) return null;
    if (value.string) return {string:bytes(value.string)};
    if (value.json) return {json:bytes(value.json.raw)};
    if (value.hash) return {hash:value.hash.map(entry=>({field:bytes(entry.field),value:bytes(entry.value)})).sort((first,second)=>first.field.data.localeCompare(second.field.data))};
    if (value.list) return {list:value.list.map(entry=>({index:entry.index,value:bytes(entry.value)})).sort((first,second)=>first.index.localeCompare(second.index))};
    if (value.set) return {set:value.set.map(bytes).sort((first,second)=>first.data.localeCompare(second.data))};
    if (value.zset) return {zset:value.zset.map(entry=>({member:bytes(entry.member),score:entry.score})).sort((first,second)=>first.member.data.localeCompare(second.member.data))};
    if (value.stream) return {stream:value.stream.map(entry=>({id:entry.id,fields:entry.fields.map(pair=>({field:bytes(pair.field),value:bytes(pair.value)}))}))};
    return {native:value.native || null};
  }
  function changed() { return !!current && (JSON.stringify(payload(current)) !== JSON.stringify(payload(draft)) || !!ttlDraft || !!invalid); }
  function editable() { return write && !!current?.editable && !pending && !loading; }
  function updateActions() {
    const dirty = changed();
    let changeCount=dirty?1:0;
    if(dirty&&!invalid){try{changeCount=operations().reduce((total,operation)=>total+Math.max(1,(operation.hash||operation.fields||operation.list||operation.members||operation.sortedSet||operation.stream||operation.ids||[]).length),0);}catch{}}
    if(query('[data-redis-change-count]'))query('[data-redis-change-count]').textContent=String(changeCount);
    const selectionBar=query('[data-redis-entry-selection]');if(selectionBar)selectionBar.hidden=!selectedEntries.size||view==='raw';
    if(query('[data-redis-entry-count]'))query('[data-redis-entry-count]').textContent=selectedEntries.size===1?'1 entry picked':selectedEntries.size+' entries picked';
    query('[data-redis-copy-name]').disabled=!current?.exists||pending||loading;
    query('[data-redis-save]')?.toggleAttribute('disabled',!editable() || !dirty || !!invalid);
    query('[data-redis-discard]')?.toggleAttribute('disabled',!dirty || pending || loading);
    for(const hook of ['rename','copy-key','delete','change-ttl']) query('[data-redis-'+hook+']')?.toggleAttribute('disabled',!editable() || !current?.version);
    query('[data-redis-add-entry]')?.toggleAttribute('disabled',!editable() || !!invalid || !['hash','list','set','zset','stream'].includes(current?.type));
    if(query('[data-redis-list-actions]'))query('[data-redis-list-actions]').hidden=!editable()||current?.type!=='list';
    query('[data-redis-reload-key]').disabled = !current || pending || loading;
    query('[data-redis-value-more]').disabled = pending || loading;
    for(const control of editor.querySelectorAll('input,textarea')) {if(control.type==='checkbox')control.disabled=!editable();else control.readOnly=!editable();}
    query('[data-redis-entry-delete]')?.toggleAttribute('disabled',pending||loading);query('[data-redis-entry-clear]')?.toggleAttribute('disabled',pending||loading);
    for(const button of editor.querySelectorAll('button')) button.disabled=!editable();
    const state=query('[data-redis-draft-state]');
    if(loading) state.textContent='Loading key… The existing draft is retained until a snapshot arrives.';
    else if(invalid) state.textContent=invalid+' Your draft is retained; fix the input before saving or changing view.';
    else if(!write) state.textContent='Read-only access. Changing view never changes Redis.';
    else if(!current?.editable) state.textContent='This value is read-only because its bounded snapshot is partial or its native type cannot be edited.';
    else state.textContent=dirty?'Unsaved changes · changing view does not save them.':'No unsaved changes.';
  }
  async function leaveDraft() {
    if(pending || loading) throw new Error('Wait for the current Redis request to finish before changing keys.');
    if(!changed()) return true;
    return confirm('Discard unsaved changes?','This action opens a different key or snapshot. Your current draft has not been saved.',()=>{},'Discard draft');
  }
  function renderTree() {
    query('[data-redis-loaded]').textContent=keyMap.size+(lastDatabaseSize?' / '+lastDatabaseSize:'')+' keys';
    query('[data-redis-loaded]').title=keyMap.size+' loaded · '+(scanComplete?'scan complete':scanBound?'loaded-list limit reached; refine the search':'scan incomplete; load more keys');
    const tree=query('[data-redis-tree]'); tree.replaceChildren();
    const filter='';
    const separator=':';
    const groups=new Map();
    const keys=[...keyMap.values()].sort((left,right)=>label(left.key).localeCompare(label(right.key)));
    for(const summary of keys) {
      const shown=label(summary.key);
      if(filter && !shown.toLocaleLowerCase().includes(filter)) continue;
      let parent=tree, prefix='';
      let parts=[];
      if(separator && summary.key.text != null) parts=shown.split(separator).slice(0,12);
      if(parts.length>1) for(const part of parts.slice(0,-1)) {
        prefix+=separator+part;
        if(!groups.has(prefix)) {
          const group=element('details'); group.open=true;
          const heading=element('summary',part || '(empty prefix)','rail-group'); group.append(heading); parent.append(group); groups.set(prefix,group);
        }
        parent=groups.get(prefix);
      }
      const button=element('button',null,'rail-item'+(current?.key.data===summary.key.data?' is-current':''));
      button.type='button';button.title=shown;button.setAttribute('aria-pressed',String(current?.key.data===summary.key.data));
      const icon=document.createElementNS('http://www.w3.org/2000/svg','svg');icon.setAttribute('viewBox','0 0 24 24');const path=document.createElementNS('http://www.w3.org/2000/svg','path');path.setAttribute('d','m15.5 7.5 3 3L22 7l-3-3-3.5 3.5ZM11 11l4.5-3.5M2 17a5 5 0 1 0 10 0 5 5 0 0 0-10 0Z');icon.append(path);
      button.append(icon,element('span',parts.length>1?parts.at(-1):shown,'rail-name'),element('span',summary.type==='ReJSON-RL'?'JSON':summary.type,'badge is-outline'));
      button.addEventListener('click',safely(async()=>{if(await leaveDraft()) await loadKey(summary.key);}));
      parent.append(button);
    }
    if(!tree.children.length) tree.append(element('div',keyMap.size?'No loaded keys match.':'No keys match. Continue scanning if the cursor is not complete.','rail-empty'));
  }
  let rescanRequested=false;
  function scanFilters() {
    const value=scanForm.elements.match.value;
    const format=scanForm.elements.matchEncoding.value;
    const pattern=!value?'*':format==='text'?'*'+value.replace(/[\\*?\[\]]/g,'\\$&')+'*':value;
    const result={matchBytes:textBytes(pattern,!value?'text':format),type:scanForm.elements.type.value};
    const count=(result.type?1:0)+(format!=='text'?1:0);query('[data-redis-filter-count]').textContent=String(count);query('[data-redis-filter-count]').hidden=!count;
    return result;
  }
  async function scan(reset=false) {
    if(scanning) {if(reset)rescanRequested=true;return;}
    scanning=true;query('[data-redis-load-more]').disabled=true;
    try {
      const filters = reset || !activeScan ? scanFilters() : activeScan;
      const result=await request('/keys/scan',{...filters,cursor:reset?'0':scanCursor,limit:200,loadedKeys:reset?[]:[...keyMap.keys()]});
      activeScan=filters;
      if(reset) {keyMap.clear();scanBound=false;}
      for(const summary of result.keys) keyMap.set(summary.key.data,summary);
      scanCursor=result.cursor;scanComplete=result.complete;
      scanBound=keyMap.size>=10000 || [...keyMap.keys()].reduce((total,key)=>total+key.length,0)>=1048576;
      lastDatabaseSize=result.databaseSize;
      renderTree();
    } catch(problem) {
      if(!keyMap.size)query('[data-redis-tree]').replaceChildren(element('div','Keys could not be loaded. Reload the list to retry.','rail-empty'));
      throw problem;
    } finally {scanning=false;query('[data-redis-load-more]').disabled=scanComplete || scanBound;if(rescanRequested){rescanRequested=false;safely(()=>scan(true))();}}
  }
  async function loadKey(key,cursor='') {
    if(!closeCell())return;
    const serial=++displaySerial;
    if(!current)query('[data-redis-key-empty]').replaceChildren(element('strong','Loading value…'),element('span','Reading a bounded snapshot.'));
    loading=true;updateActions();
    try {
      const value=await request('/keys/read',{key:bytes(key),options:{cursor,limit:500,maxBytes:262144}});
      if(serial!==displaySerial) return;
      loading=false;current=value;draft=clone(value);ttlDraft=null;selectedEntries.clear();clearInvalid();
      query('[data-redis-key-empty]').hidden=value.exists;
      query('[data-redis-value]').hidden=!value.exists;
      if(!value.exists) {
        keyMap.delete(key.data);query('[data-redis-key-empty]').replaceChildren(element('h2','Key no longer exists'),element('p','It was deleted or expired. Refresh the key scan to update the list.'));renderTree();return;
      }
      renderValue();renderTree();
    } catch(problem) {
      if(!current)query('[data-redis-key-empty]').replaceChildren(element('strong','Value unavailable'),element('span','Reload the list and choose the key again.'));
      throw problem;
    } finally {if(serial===displaySerial){loading=false;updateActions();}}
  }
  function display(value,format=encoding) {
    try {
      const text = displayBytes(value,format);
      if(format==='text' && text.includes('\r')) return {text:value.data,format:'base64'};
      return {text,format};
    } catch {return {text:value.data,format:'base64'};}
  }
  function closeCell(keep=true) {
    if(!editingCell)return true;
    if(keep&&invalid){error(invalid+' Fix or cancel this edit first.');return false;}
    const open=editingCell;editingCell=null;open.finish(keep);return true;
  }
  function moveCell(cell,down,across) {
    const row=cell.parentElement,rows=[...cell.closest('tbody').children];const targetRow=rows[rows.indexOf(row)+down];if(!targetRow)return;
    const cells=[...row.querySelectorAll('[data-redis-cell]')],targets=[...targetRow.querySelectorAll('[data-redis-cell]')];const target=targets[cells.indexOf(cell)+across];if(target)target.focus();
  }
  function editableText(text,onChange,parse,readOnly=false,initiallyChanged=false,original={text,value:parse(text)}) {
    const wrapper=element('span',null,'cell-body'),shown=element('span',text,'cell-text');wrapper.append(shown);
    let currentText=text,currentValue=parse(text);
    requestAnimationFrame(()=>{
      const cell=wrapper.closest('td');if(!cell)return;cell.classList.add('cell');cell.classList.toggle('is-changed',initiallyChanged);
      if(!editable()||readOnly)return;
      cell.dataset.redisCell='';cell.tabIndex=-1;
      function begin(){
        if(editingCell?.cell===cell)return;if(!closeCell())return;
        const beforeText=currentText,beforeValue=clone(currentValue),beforeChanged=cell.classList.contains('is-changed');
        const popup=element('div',null,'explorer-grid-editor');popup.setAttribute('role','dialog');popup.setAttribute('aria-label','Edit '+(cell.dataset.cell||'value'));
        popup.append(element('strong',cell.dataset.cell||'Value','label'));
        const input=element('textarea',null,'input');input.value=currentText;input.rows=3;input.spellcheck=false;input.setAttribute('aria-label','Edit '+(cell.dataset.cell||'value'));popup.append(input,element('span','Ctrl+Enter applies the edit. Escape cancels.','field-note'));
        const actions=element('div',null,'actions');
        for(const [label,keep] of [['Cancel',false],['Apply',true]]){const button=element('button',label,'button is-small'+(keep?'':' is-ghost'));button.type='button';button.addEventListener('click',()=>{if(closeCell(keep))cell.focus();});actions.append(button);}popup.append(actions);
        cell.classList.add('is-editing');document.body.append(popup);
        const position=()=>{const bounds=cell.getBoundingClientRect();const width=Math.min(360,innerWidth-32);popup.style.width=width+'px';popup.style.left=Math.max(16,Math.min(bounds.left,innerWidth-width-16))+'px';const height=popup.getBoundingClientRect().height;popup.style.top=Math.max(16,Math.min(bounds.bottom+4+height<=innerHeight?bounds.bottom+4:bounds.top-height-4,innerHeight-height-16))+'px';};
        editingCell={cell,popup,position,finish:keep=>{if(!keep){currentText=beforeText;currentValue=beforeValue;onChange(clone(beforeValue));cell.classList.toggle('is-changed',beforeChanged);}shown.textContent=currentText;setInvalid(input,'');cell.classList.remove('is-editing');popup.remove();updateActions();}};
        input.addEventListener('input',()=>{try{const value=parse(input.value);onChange(value);currentText=input.value;currentValue=value;setInvalid(input,'');cell.classList.toggle('is-changed',initiallyChanged||currentText!==text);}catch(problem){setInvalid(input,problem.message);}updateActions();});
        popup.addEventListener('keydown',event=>{event.stopPropagation();if(event.key==='Escape'||event.key==='Enter'&&(event.ctrlKey||event.metaKey)){event.preventDefault();if(closeCell(event.key!=='Escape'))cell.focus();}});
        position();input.focus();input.select();
      }
      cell.addEventListener('click',event=>{if(!event.target.closest('button,a,input,select,textarea'))cell.focus();});
      cell.addEventListener('dblclick',event=>{if(!event.target.closest('button,a,input,select,textarea'))begin();});
      cell.addEventListener('focus',()=>{for(const other of editor.querySelectorAll('[data-redis-cell]')){other.tabIndex=other===cell?0:-1;other.classList.toggle('is-focused',other===cell);}});
      cell.addEventListener('keydown',event=>{if(editingCell)return;if(event.key==='Enter'||event.key==='F2'){event.preventDefault();begin();}else if(event.key==='ArrowDown'||event.key==='ArrowUp'){event.preventDefault();moveCell(cell,event.key==='ArrowDown'?1:-1,0);}else if(['ArrowRight','ArrowLeft','Tab'].includes(event.key)){event.preventDefault();moveCell(cell,0,event.key==='ArrowLeft'||event.shiftKey?-1:1);}else if(event.key==='Escape'){onChange(clone(original.value));currentText=original.text;currentValue=clone(original.value);shown.textContent=original.text;cell.classList.remove('is-changed');renderEditor();updateActions();}});
      if(!editor.querySelector('[data-redis-cell][tabindex="0"]'))cell.tabIndex=0;
    });
    return wrapper;
  }
  function byteInput(value,onChange,readOnly=false,initiallyChanged=false,originalValue=value) {
    const rendered=display(value);
    const wrapper=editableText(rendered.text,onChange,text=>text===rendered.text?clone(value):textBytes(text,rendered.format),readOnly,initiallyChanged,{text:display(originalValue).text,value:clone(originalValue)});
    if(rendered.format!==encoding)wrapper.title='Binary data · base64';
    return wrapper;
  }
  document.addEventListener('pointerdown',event=>{if(editingCell&&!editingCell.popup.contains(event.target)&&!editingCell.cell.contains(event.target))closeCell();});
  window.addEventListener('resize',()=>editingCell?.position());window.addEventListener('scroll',()=>editingCell?.position(),true);
  function gridTable(headers,rows) {
    const output=table(headers,rows);output.classList.add('is-grid');
    for(const row of output.querySelectorAll('tbody > tr'))[...row.children].forEach((cell,index)=>{cell.classList.add('cell');cell.dataset.cell=headers[index]||'Value';if(!cell.children.length){const shown=element('span',cell.textContent,'cell-text');cell.replaceChildren(shown);}});
    return output;
  }
  function collectionTable(headers,rows,indices=[]) {
    if(!write&&headers.at(-1)==='Action')return gridTable(headers.slice(0,-1),rows.map(row=>row.slice(0,-1)));
    const output=gridTable(headers,rows);
    if(headers.at(-1)==='Action'){output.querySelector('thead tr').lastElementChild.textContent='';output.querySelector('thead tr').lastElementChild.classList.add('table-pick');for(const row of output.querySelectorAll('tbody > tr'))row.lastElementChild.classList.add('table-pick');}
    const selectable=editable()&&indices.length&&['hash','set','zset','stream'].includes(current.type);
    if(selectable){
      const heading=element('th',null,'table-pick');const all=query('[data-redis-checkbox-template]').content.firstElementChild.cloneNode(true);const every=all.querySelector('input');every.setAttribute('aria-label','Pick every entry on this page');heading.append(all);output.querySelector('thead tr').prepend(heading);
      const boxes=[];
      [...output.querySelectorAll('tbody > tr')].forEach((row,index)=>{const picked=element('td',null,'table-pick');const control=query('[data-redis-checkbox-template]').content.firstElementChild.cloneNode(true);const box=control.querySelector('input');box.checked=selectedEntries.has(indices[index]);box.setAttribute('aria-label','Pick this entry');picked.append(control);row.prepend(picked);boxes.push({box,row,index:indices[index]});});
      const refresh=()=>{for(const item of boxes){if(item.box.checked)selectedEntries.add(item.index);else selectedEntries.delete(item.index);item.row.classList.toggle('is-picked',item.box.checked);}const count=boxes.filter(item=>item.box.checked).length;every.checked=count===boxes.length;every.indeterminate=count>0&&count<boxes.length;updateActions();};
      every.addEventListener('change',()=>{for(const item of boxes)item.box.checked=every.checked;refresh();});for(const item of boxes)item.box.addEventListener('change',refresh);refresh();
    }
    return output;
  }
  function actionButton(text,operation,disabled=false) {
    const destructive=/^(Remove|Delete)/.test(text);
    const button=element('button',destructive?null:text,destructive?'step is-bare':'button is-ghost is-small');button.type='button';button.disabled=disabled || !editable();
    if(destructive){button.setAttribute('aria-label','Delete this entry');button.title='Delete this entry';const icon=document.createElementNS('http://www.w3.org/2000/svg','svg');icon.setAttribute('viewBox','0 0 24 24');const path=document.createElementNS('http://www.w3.org/2000/svg','path');path.setAttribute('d','M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6');icon.append(path);button.append(icon);}
    button.addEventListener('click',safely(()=>{if(invalid)throw new Error(invalid+' Fix or discard invalid input first.');return operation();}));return button;
  }
  async function deleteEntries(indices) {
    if(changed())throw new Error('Save or discard staged edits before deleting entries.');
    const kind=current.type;if(!['hash','set','zset','stream'].includes(kind))return;
    await confirm(indices.length===1?'Delete this entry?':'Delete the entries you picked?','These entries are removed from '+label(current.key)+'. This cannot be undone.',async()=>{const removed=new Set(indices);draft[kind]=draft[kind].filter((entry,index)=>!removed.has(index));await save();selectedEntries.clear();updateActions();},indices.length===1?'Delete entry':'Delete entries');
  }
  function removeEntry(array,index) {return deleteEntries([index]);}
  function visibleRows(rows,values) {
    const filter=query('[data-redis-value-filter]').value.toLocaleLowerCase();
    return rows.map((entry,index)=>({entry,index})).filter(item=>!filter || values(item.entry).toLocaleLowerCase().includes(filter));
  }
  function manifest(value) {
    const encoded=item=>{const rendered=display(item);return {encoding:rendered.format,data:rendered.text};};
    if(value.hash) return {hash:value.hash.map(entry=>({field:encoded(entry.field),value:encoded(entry.value)}))};
    if(value.list) return {list:value.list.map(entry=>({index:entry.index,value:encoded(entry.value)}))};
    if(value.set) return {set:value.set.map(encoded)};
    if(value.zset) return {zset:value.zset.map(entry=>({member:encoded(entry.member),score:entry.score}))};
    if(value.stream) return {stream:value.stream.map(entry=>({id:entry.id,fields:entry.fields.map(pair=>({field:encoded(pair.field),value:encoded(pair.value)}))}))};
    return {native:value.native || null};
  }
  function parseManifest(raw) {
    selectedEntries.clear();
    const parsed=JSON.parse(raw), kind=current.type==='ReJSON-RL'?'json':current.type;
    const decode=item=>{if(!item || typeof item.data!=='string' || !['text','hex','base64'].includes(item.encoding)) throw new Error('Every byte field needs a text, hex or base64 encoding and string data.');return textBytes(item.data,item.encoding);};
    if(!parsed || Object.keys(parsed).length!==1 || !Array.isArray(parsed[kind])) throw new Error('Keep the '+kind+' array in this raw envelope.');
    const result=parsed[kind];if(result.length>1000) throw new Error('At most 1,000 entries can be edited in one snapshot.');
    if(kind==='hash') draft.hash=result.map(entry=>({field:decode(entry.field),value:decode(entry.value)}));
    if(kind==='list') draft.list=result.map(entry=>{if(typeof entry.index!=='string'||!/^(0|[1-9][0-9]*)$/.test(entry.index)) throw new Error('List indices must be decimal strings.');return {index:entry.index,value:decode(entry.value)};});
    if(kind==='set') draft.set=result.map(decode);
    if(kind==='zset') draft.zset=result.map(entry=>{if(typeof entry.score!=='string') throw new Error('Scores must remain strings.');return {member:decode(entry.member),score:entry.score};});
    if(kind==='stream') draft.stream=result.map(entry=>{if(typeof entry.id!=='string'||!Array.isArray(entry.fields)) throw new Error('Stream IDs must be strings and fields must be an array.');return {id:entry.id,fields:entry.fields.map(pair=>({field:decode(pair.field),value:decode(pair.value)}))};});
  }
  function renderEditor() {
    if(!closeCell())return;
    editor.replaceChildren();
    if(!draft) return;
    query('[data-redis-filter-label]').hidden=!!draft.string || !!draft.json || view==='raw';
    if(draft.string || draft.json) {
      const original=draft.string || draft.json.raw;
      let rendered=display(original), displayed=rendered.text;
      if(view==='pretty' && rendered.format==='text') {try{displayed=prettyJSON(displayed);}catch{}}
      const textarea=element('textarea',null,'redis-code');textarea.value=displayed;textarea.readOnly=!editable();textarea.spellcheck=false;
      textarea.setAttribute('aria-label',(!draft.string && draft.json?'JSON':'String')+' value · '+rendered.format);
      textarea.addEventListener('input',()=>{
        try {
          const value=textarea.value===displayed?clone(original):textBytes(textarea.value,rendered.format);
          if(draft.string) draft.string=value; else {JSON.parse(displayBytes(value));draft.json.raw=value;}
          setInvalid(textarea, '');
        }catch(problem){setInvalid(textarea, problem.message);}
        updateActions();
      });editor.append(textarea);
      if(rendered.format!==encoding) editor.append(element('p','These bytes cannot be edited losslessly in a text control (binary data or carriage returns). They are shown as base64. Choose Hex or Base64 to inspect the exact bytes.','field-note'));
      return;
    }
    if(view==='raw') {
      const raw=JSON.stringify(manifest(draft),null,2),textarea=element('textarea',null,'redis-code');textarea.value=raw;textarea.readOnly=!editable();textarea.spellcheck=false;textarea.setAttribute('aria-label','Lossless raw collection envelope');
      textarea.addEventListener('input',()=>{try{parseManifest(textarea.value);setInvalid(textarea, '');}catch(problem){setInvalid(textarea, problem.message);}updateActions();});
      editor.append(element('p','Lossless collection envelope. Indices, stream IDs, scores and byte data are strings. Changing encoding never writes data.','field-note'),textarea);return;
    }
    if(draft.hash) {
      const rows=visibleRows(draft.hash,entry=>label(entry.field)+' '+label(entry.value));
      editor.append(collectionTable(['Field','Value','Action'],rows.map(({entry,index})=>[byteInput(entry.field,value=>entry.field=value,false,!current.hash.some(original=>equalBytes(original.field,entry.field)),entryOrigin('hash',entry).field),byteInput(entry.value,value=>entry.value=value,false,!equalBytes(current.hash.find(original=>equalBytes(original.field,entry.field))?.value,entry.value),entryOrigin('hash',entry).value),actionButton('Remove',()=>removeEntry(draft.hash,index))]),rows.map(row=>row.index)));
    } else if(draft.list) {
      const rows=visibleRows(draft.list,entry=>entry.index+' '+label(entry.value));
      editor.append(gridTable(['Index','Value'],rows.map(({entry})=>[entry.index,byteInput(entry.value,value=>entry.value=value,false,!equalBytes(current.list.find(original=>original.index===entry.index)?.value,entry.value),current.list.find(original=>original.index===entry.index)?.value||entry.value)])));
    } else if(draft.set) {
      const rows=visibleRows(draft.set,label);
      editor.append(collectionTable(['Member','Action'],rows.map(({entry,index})=>[byteInput(entry,value=>{entryOrigins.set(value,entryOrigin('set',entry));draft.set[index]=value;},false,!current.set.some(original=>equalBytes(original,entry)),entryOrigin('set',entry)),actionButton('Remove',()=>removeEntry(draft.set,index))]),rows.map(row=>row.index)));
    } else if(draft.zset) {
      const rows=visibleRows(draft.zset,entry=>label(entry.member)+' '+entry.score);
      editor.append(collectionTable(['Member','Score','Action'],rows.map(({entry,index})=>{
        const input=editableText(entry.score,value=>entry.score=value,value=>value,false,current.zset.find(original=>equalBytes(original.member,entry.member))?.score!==entry.score,{text:entryOrigin('zset',entry).score,value:entryOrigin('zset',entry).score});
        return [byteInput(entry.member,value=>entry.member=value,false,!current.zset.some(original=>equalBytes(original.member,entry.member)),entryOrigin('zset',entry).member),input,actionButton('Remove',()=>removeEntry(draft.zset,index))];
      }),rows.map(row=>row.index)));
    } else if(draft.stream) {
      const rows=visibleRows(draft.stream,entry=>entry.id+' '+entry.fields.map(pair=>label(pair.field)+' '+label(pair.value)).join(' '));
      editor.append(collectionTable(['Stream ID','Fields','Action'],rows.map(({entry,index})=>{
        const fields=table(['Field','Value'],entry.fields.map(pair=>[element('pre',label(pair.field)),element('pre',label(pair.value))]));
        return [entry.id,fields,actionButton('Delete entry',()=>removeEntry(draft.stream,index))];
      }),rows.map(row=>row.index)));
    } else if(draft.native) {
      editor.append(table(['Property','Value'],(draft.native.summary||[]).map(property=>[property.name,property.value])));
      if(draft.native.serialized) editor.append(element('pre',display(draft.native.serialized).text,'redis-code'));
      editor.append(element('p','This native type is inspection-only. No replacement is inferred from its summary.','field-note'));
    } else editor.append(element('p','No value entries were returned.','field-note'));
  }
  function expiryDuration(milliseconds) {
    let seconds = (BigInt(milliseconds) + 999n) / 1000n;
    const parts = [];
    for (const [size, label] of [[86400n, 'd'], [3600n, 'h'], [60n, 'm'], [1n, 's']]) {
      const amount = seconds / size; seconds %= size;
      if (amount || !parts.length && size === 1n) parts.push(amount + label);
      if (parts.length === 2) break;
    }
    return parts.join(' ');
  }
  function renderTTL() {
    const ttl = ttlDraft || current.ttl;
    const target = query('[data-redis-ttl]');
    target.textContent = (ttlDraft ? 'Staged · ' : '') + (ttl.mode === 'persistent' ? 'No expiry' : ttl.mode === 'missing' ? 'Key missing' : ttl.mode === 'relative' ? expiryDuration(ttl.milliseconds) : ttlDraft ? 'Expiry date set' : expiryDuration(current.ttl.milliseconds));
    const expiry = ttl.expiresAtMilliseconds || current.ttl.expiresAtMilliseconds;
    const date = new Date(Number(expiry));
    target.title = expiry && !Number.isNaN(date.getTime()) ? 'Expires ' + date.toLocaleString() : 'Change key expiry';
  }
  function renderValue() {
    query('[data-redis-key-name]').textContent=label(current.key);query('[data-redis-type]').textContent=current.type;
    query('[data-redis-copy-name]').textContent=current.key.text == null?'Copy key name · base64':'Copy key name';
    query('[data-redis-type]').hidden=false;
    renderTTL();
    query('[data-redis-length]').textContent='Length '+current.length;
    const partial=query('[data-redis-partial]');partial.hidden=!!current.complete;
    partial.textContent='Partial bounded snapshot. Editing, rename, copy and deletion are disabled because a safe complete version reference is unavailable. Continue through the available pages to inspect the value.';
    query('[data-redis-value-more]').hidden=!current.cursor || current.complete;
    renderEditor();updateActions();
    const groups=query('[data-redis-stream-groups]');groups.replaceChildren();
    if(current.groups?.length) {groups.append(element('h3','Consumer groups'),table(['Group','Consumers','Pending','Last delivered ID','Entries read','Lag'],current.groups.map(group=>[label(group.name),group.consumers,group.pending,group.lastDeliveredId,group.entriesRead,group.lag])));}
  }
  function validateDraft() {
    if(invalid) throw new Error(invalid);
    if(draft.hash && new Set(draft.hash.map(entry=>entry.field.data)).size!==draft.hash.length) throw new Error('Hash fields must be unique.');
    if(draft.set && new Set(draft.set.map(entry=>entry.data)).size!==draft.set.length) throw new Error('Set members must be unique.');
    if(draft.zset && new Set(draft.zset.map(entry=>entry.member.data)).size!==draft.zset.length) throw new Error('Sorted-set members must be unique.');
    if(draft.stream) {
      const original=new Map(current.stream.map(entry=>[entry.id,entry]));
      const fields = entry => entry.fields.map(pair => ({field:bytes(pair.field),value:bytes(pair.value)}));
      for(const entry of draft.stream) if(original.has(entry.id) && JSON.stringify(fields(entry))!==JSON.stringify(fields(original.get(entry.id)))) throw new Error('Existing stream entries are immutable. Delete one or add a new entry instead.');
    }
  }
  function operations() {
    validateDraft();const result=[];
    if(draft.string && !equalBytes(draft.string,current.string)) result.push({operation:'string_set',value:bytes(draft.string)});
    if(!draft.string && draft.json && !equalBytes(draft.json.raw,current.json.raw)) result.push({operation:'json_set',json:bytes(draft.json.raw)});
    if(draft.hash) {
      const original=new Map(current.hash.map(entry=>[entry.field.data,entry.value]));const desired=new Set(draft.hash.map(entry=>entry.field.data));
      const changes=draft.hash.filter(entry=>!equalBytes(original.get(entry.field.data),entry.value));
      if(changes.length) result.push({operation:'hash_set',hash:changes.map(entry=>({field:bytes(entry.field),value:bytes(entry.value)}))});
      const removed=current.hash.filter(entry=>!desired.has(entry.field.data)).map(entry=>bytes(entry.field));if(removed.length) result.push({operation:'hash_delete',fields:removed});
    }
    if(draft.list) {
      const desired=new Map(draft.list.map(entry=>[entry.index,entry]));
      for(const entry of current.list) if(!desired.has(entry.index)) throw new Error('List rows cannot be removed from the raw envelope. Use Pop first / Pop last, or edit their values.');
      const additions=draft.list.filter(entry=>!current.list.some(original=>original.index===entry.index));
      if(additions.length) {
        const start=BigInt(current.length);
        if(additions.some((entry,index)=>entry.index!==(start+BigInt(index)).toString())) throw new Error('New list entries must have consecutive string indices after the current length.');
        result.push({operation:'list_push_right',list:additions.map(entry=>bytes(entry.value))});
      }
      for(const entry of draft.list) {const original=current.list.find(original=>original.index===entry.index);if(original&&!equalBytes(entry.value,original.value)) result.push({operation:'list_set',index:entry.index,value:bytes(entry.value)});}
    }
    if(draft.set) {
      const original=new Set(current.set.map(entry=>entry.data)),desired=new Set(draft.set.map(entry=>entry.data));
      const added=draft.set.filter(entry=>!original.has(entry.data));if(added.length) result.push({operation:'set_add',members:added.map(bytes)});
      const removed=current.set.filter(entry=>!desired.has(entry.data));if(removed.length) result.push({operation:'set_remove',members:removed.map(bytes)});
    }
    if(draft.zset) {
      const original=new Map(current.zset.map(entry=>[entry.member.data,entry.score])),desired=new Set(draft.zset.map(entry=>entry.member.data));
      const added=draft.zset.filter(entry=>original.get(entry.member.data)!==entry.score);if(added.length) result.push({operation:'zset_add',sortedSet:added.map(entry=>({member:bytes(entry.member),score:entry.score}))});
      const removed=current.zset.filter(entry=>!desired.has(entry.member.data));if(removed.length) result.push({operation:'zset_remove',members:removed.map(entry=>bytes(entry.member))});
    }
    if(draft.stream) {
      const original=new Set(current.stream.map(entry=>entry.id)),desired=new Set(draft.stream.map(entry=>entry.id));
      const added=draft.stream.filter(entry=>!original.has(entry.id));
      for(const entry of added) result.push({operation:'stream_add',stream:[{id:entry.id,fields:entry.fields.map(pair=>({field:bytes(pair.field),value:bytes(pair.value)}))}],_entry:entry});
      const removed=current.stream.filter(entry=>!desired.has(entry.id));if(removed.length) result.push({operation:'stream_delete',ids:removed.map(entry=>entry.id)});
    }
    if(ttlDraft) {if(!result.length)result.push({operation:'ttl'});result[0].ttl=clone(ttlDraft);}
    return result;
  }
  async function save() {
    if(!closeCell())return;
    if(pending||!editable())return;
    const actions=operations();if(!actions.length)return;
    pending=true;renderEditor();updateActions();
    try {
      const mutations=actions.map(({_entry,...mutation})=>({...mutation,key:bytes(current.key),version:current.version,readOptions:{limit:500,maxBytes:262144}}));
      current=await request('/keys/batch',{mutations});
      draft=clone(current);ttlDraft=null;selectedEntries.clear();clearInvalid();
      if(!current.exists){query('[data-redis-value]').hidden=true;query('[data-redis-key-empty]').hidden=false;query('[data-redis-key-empty]').replaceChildren(element('h2','Key removed'),element('p','Removing the final collection member removed the Redis key.'));keyMap.delete(current.key.data);}
      else {keyMap.set(current.key.data,{key:current.key,type:current.type});renderValue();}
      renderTree();
    } catch(problem) {
      problem.message+=' The staged draft is retained. If Redis reported an execution error, inspect the current key before retrying; commands in a transaction do not roll back execution errors.';
      throw problem;
    } finally {pending=false;renderEditor();updateActions();}
  }
  async function keyOperation(operation) {
    if(pending||loading)throw new Error('Wait for the current Redis request to finish.');
    pending=true;updateActions();
    try{return await operation();}finally{pending=false;renderEditor();updateActions();}
  }
  function pushFirst() {
    if(changed()) throw new Error('Save or discard staged edits before pushing a first list value.');
    return dialog({title:'Push first list value',description:'This immediately prepends one value to the list. Existing indices move right.',fields:[{name:'encoding',label:'Encoding',options:['text','hex','base64']},{name:'value',label:'Value',type:'textarea'}],submit:'Push first',onSubmit:values=>keyOperation(async()=>{
      const result=await request('/keys/mutate',{operation:'list_push_left',key:bytes(current.key),version:current.version,list:[textBytes(values.value,values.encoding)],readOptions:{limit:500,maxBytes:262144}});
      current=result;draft=clone(result);selectedEntries.clear();renderValue();
    })});
  }
  function popList(side) {
    if(changed())throw new Error('Save or discard staged edits before popping a list entry.');
    return confirm('Pop '+(side==='left'?'first':'last')+' list entry?','This immediately removes one value from the list.',()=>keyOperation(async()=>{
      const result=await request('/keys/mutate',{operation:side==='left'?'list_pop_left':'list_pop_right',key:bytes(current.key),version:current.version,readOptions:{limit:500,maxBytes:262144}});
      current=result;draft=clone(result);ttlDraft=null;selectedEntries.clear();if(result.exists)renderValue();else await loadKey(result.key);
    }),'Pop entry');
  }
  const creationForm=query('[data-redis-create-form]');
  function creationFields() {
    if(!creationForm)return;
    const kind=creationForm.elements.type.value;
    for(const [selector,visible] of [['field',['hash','stream'].includes(kind)],['score',kind==='zset'],['stream',kind==='stream'],['stream-fields',kind==='stream']])query('[data-redis-create-'+selector+']').hidden=!visible;
    query('[data-redis-create-value-label]').textContent=['set','zset'].includes(kind)?'Member':'Value';
    if(kind==='ReJSON-RL'&&!creationForm.elements.value.value)creationForm.elements.value.value='{}';
    const mode=creationForm.elements.expiryMode.value;
    root.querySelectorAll('[data-redis-create-relative]').forEach(node=>node.hidden=mode!=='relative');query('[data-redis-create-absolute]').hidden=mode!=='absolute';
  }
  function creationMode(entry) {
    if(!creationForm)return;
    creationForm.dataset.entryMode=String(entry);
    query('[data-redis-create-identity]').hidden=entry;query('[data-redis-create-expiry-row]').hidden=entry;
    creationForm.elements.key.required=!entry;creationForm.elements.type.disabled=entry;
    query('[data-redis-create-title]').textContent=entry?'Add an entry to '+label(current.key):'Add a key';
    query('[data-redis-create-submit]').textContent=entry?'Add entry':'Add key';
    if(entry){creationForm.elements.type.value=current.type;creationForm.elements.type.dispatchEvent(new Event('change',{bubbles:true}));}
    creationFields();document.dispatchEvent(new Event('sqldash:scan'));
  }
  function creationExpiry() {
    const mode=creationForm.elements.expiryMode.value;
    if(mode==='persistent')return {mode};
    if(mode==='relative'){
      const value=creationForm.elements.duration.value;if(!/^[1-9][0-9]*$/.test(value))throw new Error('Enter a positive whole-number duration.');
      const milliseconds=BigInt(value)*BigInt(creationForm.elements.durationUnit.value);if(milliseconds>9223372036854775807n)throw new Error('The expiry duration is too large.');
      return {mode,milliseconds:milliseconds.toString()};
    }
    const moment=Date.parse(creationForm.elements.expiresAt.value);if(mode!=='absolute'||!Number.isSafeInteger(moment)||moment<=Date.now())throw new Error('Choose a future expiry date and time.');
    return {mode,expiresAtMilliseconds:String(moment)};
  }
  function creationMutation() {
    const form=creationForm,entry=form.dataset.entryMode==='true',kind=entry?current.type:form.elements.type.value;
    const value=textBytes(form.elements.value.value,form.elements.encoding.value),field=['hash','stream'].includes(kind)?textBytes(form.elements.field.value,form.elements.encoding.value):null;
    const operation=entry?{hash:'hash_set',list:'list_push_right',set:'set_add',zset:'zset_add',stream:'stream_add'}[kind]:'create';
    if(!operation)throw new Error('This value does not support adding entries.');
    const mutation={operation,key:entry?bytes(current.key):textBytes(form.elements.key.value,form.elements.keyEncoding.value),readOptions:{limit:500,maxBytes:262144}};
    if(entry)mutation.version=current.version;else {mutation.type=kind;mutation.ttl=creationExpiry();}
    if(kind==='string')mutation.value=value;
    else if(kind==='hash'){if(entry&&current.hash.some(pair=>equalBytes(pair.field,field)))throw new Error('This field already exists. Edit it in the grid instead.');mutation.hash=[{field,value}];}
    else if(kind==='list')mutation.list=[value];
    else if(kind==='set')mutation.members=[value];
    else if(kind==='zset')mutation.sortedSet=[{member:value,score:form.elements.score.value}];
    else if(kind==='stream'){
      const extra=JSON.parse(form.elements.extraFields.value||'[]');if(!Array.isArray(extra)||extra.length>999)throw new Error('Additional fields must be an array of at most 999 pairs.');
      const pairs=extra.map(pair=>{if(typeof pair.field!=='string'||typeof pair.value!=='string')throw new Error('Stream fields and values must be strings.');return {field:textBytes(pair.field,form.elements.encoding.value),value:textBytes(pair.value,form.elements.encoding.value)};});
      mutation.stream=[{id:form.elements.streamID.value,fields:[{field,value},...pairs]}];
    }else if(kind==='ReJSON-RL'){JSON.parse(displayBytes(value));mutation.json=value;}
    else throw new Error('Choose a supported key type.');
    return mutation;
  }
  creationForm?.addEventListener('submit',safely(async event=>{
    const notice=query('[data-redis-create-error]');notice.hidden=true;
    if(pending||loading)return;
    try{
      if(!await leaveDraft())return;
      const mutation=creationMutation();pending=true;query('[data-redis-create-submit]').disabled=true;updateActions();
      const result=await request('/keys/mutate',mutation);current=result;draft=clone(result);ttlDraft=null;selectedEntries.clear();clearInvalid();
      query('#redis-create').hidden=true;query('[data-redis-key-empty]').hidden=result.exists;query('[data-redis-value]').hidden=!result.exists;
      if(result.exists){keyMap.set(result.key.data,{key:result.key,type:result.type});renderValue();}else{keyMap.delete(result.key.data);query('[data-redis-key-empty]').replaceChildren(element('strong','Key expired'),element('span','Its expiry elapsed before the value could be read.'));}
      renderTree();
    }catch(problem){notice.textContent=problem.message;notice.hidden=false;}
    finally{pending=false;query('[data-redis-create-submit]').disabled=false;renderEditor();updateActions();}
  }));
  query('[data-redis-create-type]')?.addEventListener('change',creationFields);query('[data-redis-create-expiry]')?.addEventListener('change',creationFields);
  root.querySelectorAll('[data-reveal="redis-create"]').forEach(button=>button.addEventListener('click',()=>{if(!query('#redis-create').hidden)creationMode(false);}));
  creationFields();
  scanForm.addEventListener('submit',safely(()=>scan(true)));
  query('[data-redis-refresh]').addEventListener('click',safely(()=>scan(true)));
  query('[data-redis-entry-clear]')?.addEventListener('click',()=>{selectedEntries.clear();renderEditor();updateActions();});
  query('[data-redis-entry-delete]')?.addEventListener('click',safely(()=>deleteEntries([...selectedEntries])));
  let searchTimer;
  query('[data-redis-key-search]').addEventListener('input',()=>{clearTimeout(searchTimer);searchTimer=setTimeout(()=>safely(()=>scan(true))(),250);});
  query('[data-redis-clear-filters]').addEventListener('click',safely(async()=>{for(const [name,value] of [['type',''],['matchEncoding','text']]){scanForm.elements[name].value=value;scanForm.elements[name].dispatchEvent(new Event('change',{bubbles:true}));}await scan(true);}));
  query('[data-redis-load-more]').addEventListener('click',safely(()=>scan(false)));
  query('[data-redis-value-filter]').addEventListener('input',()=>{if(invalid)return;selectedEntries.clear();renderEditor();updateActions();});
  query('[data-redis-value-view]').addEventListener('change',event=>{if(invalid){event.target.value=view;error(invalid+' Fix or discard the draft before changing view.');return;}view=event.target.value;renderEditor();});
  query('[data-redis-value-encoding]').addEventListener('change',event=>{if(invalid){event.target.value=encoding;error(invalid+' Fix or discard the draft before changing encoding.');return;}encoding=event.target.value;renderEditor();});
  query('[data-redis-reload-key]').addEventListener('click',safely(async()=>{if(current&&await leaveDraft())await loadKey(current.key);}));
  query('[data-redis-value-more]').addEventListener('click',safely(async()=>{if(current&&await leaveDraft())await loadKey(current.key,current.cursor);}));
  query('[data-redis-copy-name]').addEventListener('click',safely(()=>{try{return copy(displayBytes(current.key));}catch{return copy(current.key.data);}}));
  query('[data-redis-save]')?.addEventListener('click',safely(save));
  query('[data-redis-discard]')?.addEventListener('click',safely(()=>{closeCell(false);draft=clone(current);ttlDraft=null;selectedEntries.clear();clearInvalid();renderValue();}));
  query('[data-redis-add-entry]')?.addEventListener('click',safely(()=>{if(changed())throw new Error('Save or discard staged edits before adding an entry.');creationMode(true);query('#redis-create').hidden=false;const target=['hash','stream'].includes(current.type)?creationForm.elements.field:creationForm.elements.value;target.focus();}));
  query('[data-redis-list-push]')?.addEventListener('click',safely(pushFirst));
  query('[data-redis-list-pop-first]')?.addEventListener('click',safely(()=>popList('left')));
  query('[data-redis-list-pop-last]')?.addEventListener('click',safely(()=>popList('right')));
  query('[data-redis-change-ttl]')?.addEventListener('click', safely(() => {
    const milliseconds = ttlDraft?.milliseconds || current.ttl.milliseconds || '0';
    const seconds = milliseconds.startsWith('-') ? '0' : ((BigInt(milliseconds) + 999n) / 1000n).toString();
    const units = {milliseconds: 1n, seconds: 1000n, minutes: 60000n, hours: 3600000n, days: 86400000n};
    const pendingDialog = dialog({title: 'Change expiry', description: 'Stage the expiry, then Save changes to apply it.', fields: [
      {name: 'key', label: 'Key', value: label(current.key), readOnly: true},
      {name: 'mode', label: 'Expiry', options: [{value: 'relative', label: 'After a duration'}, {value: 'persistent', label: 'No expiry'}, {value: 'absolute', label: 'At a date and time'}], value: ttlDraft?.mode || (current.ttl.mode === 'persistent' ? 'persistent' : 'relative')},
      {name: 'amount', label: 'Duration', type: 'number', value: seconds},
      {name: 'unit', label: 'Unit', options: [{value: 'seconds', label: 'Seconds'}, {value: 'minutes', label: 'Minutes'}, {value: 'hours', label: 'Hours'}, {value: 'days', label: 'Days'}, {value: 'milliseconds', label: 'Milliseconds'}], value: 'seconds'},
      {name: 'expiresAt', label: 'Expires at', type: 'datetime-local', value: ''}
    ], submit: 'Stage expiry', onSubmit: values => {
      let expiry;
      if (values.mode === 'persistent') expiry = {mode: 'persistent'};
      else if (values.mode === 'relative') {
        if (!/^[1-9][0-9]*$/.test(values.amount) || !units[values.unit]) throw new Error('Enter a positive whole duration. Choose No expiry to keep the key forever.');
        expiry = {mode: 'relative', milliseconds: (BigInt(values.amount) * units[values.unit]).toString()};
      } else {
        const moment = Date.parse(values.expiresAt);
        if (!Number.isSafeInteger(moment) || moment <= Date.now()) throw new Error('Choose a future expiry date and time.');
        expiry = {mode: 'absolute', expiresAtMilliseconds: String(moment)};
      }
      ttlDraft = expiry.mode === 'persistent' && current.ttl.mode === 'persistent' ? null : expiry;
      renderTTL(); updateActions();
    }});
    const form = query('[data-redis-dialog-form]');
    const update = () => {
      for (const name of ['amount', 'unit']) form.elements[name].closest('.field').hidden = form.elements.mode.value !== 'relative';
      form.elements.expiresAt.closest('.field').hidden = form.elements.mode.value !== 'absolute';
    };
    form.elements.mode.addEventListener('change', update); update();
    const presets = element('div', null, 'actions is-left');
    for (const [title, amount, unit] of [['Forever', '', 'seconds'], ['10 seconds', '10', 'seconds'], ['1 minute', '1', 'minutes'], ['1 hour', '1', 'hours'], ['1 day', '1', 'days']]) {
      const button = element('button', title, 'button is-outline is-small'); button.type = 'button';
      button.addEventListener('click', () => {
        form.elements.mode.value = amount ? 'relative' : 'persistent';
        form.elements.amount.value = amount; form.elements.unit.value = unit;
        form.elements.mode.dispatchEvent(new Event('change', {bubbles: true}));
        form.elements.unit.dispatchEvent(new Event('change', {bubbles: true}));
      }); presets.append(button);
    }
    query('[data-redis-dialog-body]').append(presets);
    return pendingDialog;
  }));
  async function targetDialog(rename) {
    if(changed())throw new Error('Save or discard staged edits before '+(rename?'renaming':'copying')+' this key.');
    return dialog({title:rename?'Rename key':'Copy key',fields:[{name:'target',label:'New key name',value:''},{name:'encoding',label:'Key encoding',options:['text','hex','base64']},{name:'replace',label:'If target exists',options:[{value:'no',label:'Refuse · preserve target'},{value:'yes',label:'Replace target'}]}],submit:rename?'Rename key':'Copy key',onSubmit:values=>keyOperation(async()=>{
      const target=textBytes(values.target,values.encoding),replace=values.replace==='yes';
      if(rename){await request('/keys/rename',{key:bytes(current.key),target,version:current.version,replace});keyMap.delete(current.key.data);await loadKey(target);}
      else{const result=await request('/keys/mutate',{operation:'copy',key:bytes(current.key),target,version:current.version,replace,readOptions:{limit:500,maxBytes:262144}});await loadKey(result.key);}
      selectedEntries.clear();keyMap.set(current.key.data,{key:current.key,type:current.type});renderTree();
    })});
  }
  query('[data-redis-rename]')?.addEventListener('click',safely(()=>targetDialog(true)));
  query('[data-redis-copy-key]')?.addEventListener('click',safely(()=>targetDialog(false)));
  query('[data-redis-delete]')?.addEventListener('click',safely(()=>confirm('Delete key?',label(current.key)+' will be permanently removed. Unsaved edits are not applied.',()=>keyOperation(async()=>{await request('/keys/delete',{keys:[{key:bytes(current.key),version:current.version}]});keyMap.delete(current.key.data);await loadKey(current.key);renderTree();}),'Delete key')));
  window.addEventListener('beforeunload',event=>{if(changed()){event.preventDefault();event.returnValue='';}});
  safely(()=>scan())();
})();
