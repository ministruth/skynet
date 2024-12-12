import menus from './zh-CN/menus';
import pages from './zh-CN/pages';
import tables from './zh-CN/tables';
import titles from './zh-CN/titles';

export default {
  'app.copyright.author': 'imwxz',
  'app.ok': '确认',
  'app.cancel': '取消',
  'app.confirm': '此操作无法撤销！',
  'app.filesize': '文件过大，限制大小为{size}',
  'app.loading': '加载中',

  'app.table.lastupdate': '最后更新：{time}',
  'app.table.polling.start': '拉取',
  'app.table.polling.stop': '停止',
  'app.table.createdat': '创建时间',
  'app.table.updatedat': '更新时间',
  'app.table.id': 'ID',
  'app.table.searchtext': '搜索文本',

  'app.op': '操作',
  'app.op.delete': '删除',
  'app.op.deleteall': '删除全部',
  'app.op.add': '添加',
  'app.op.clone': '克隆',
  'app.op.update': '更新',
  'app.op.upload': '上传',
  ...pages,
  ...tables,
  ...titles,
  ...menus,
};
