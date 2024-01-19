import menus from './en-US/menus';
import pages from './en-US/pages';
import titles from './en-US/titles';

export default {
  'app.copyright.author': 'imwxz',
  'app.ok': 'OK',
  'app.cancel': 'Cancel',
  'app.confirm': 'This operation cannot be undone!',
  'app.filesize': 'File is too large, should be less than {size}',

  'app.table.lastupdate': 'Last update: {time}',
  'app.table.polling.start': 'Poll',
  'app.table.polling.stop': 'Stop',
  'app.table.createdat': 'Created At',
  'app.table.updatedat': 'Updated At',
  'app.table.id': 'ID',
  'app.table.searchtext': 'Text',

  'app.op': 'Operation',
  'app.op.delete': 'Delete',
  'app.op.deleteall': 'Delete All',
  'app.op.add': 'Add',
  'app.op.delete.tip': 'Delete',
  ...pages,
  ...titles,
  ...menus,
};
