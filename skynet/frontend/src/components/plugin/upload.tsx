import { fileToBase64, getIntl } from '@/utils';
import { UploadOutlined } from '@ant-design/icons';
import { Button, Upload, message } from 'antd';
import { UploadChangeParam, UploadFile } from 'antd/es/upload';

export interface PluginUploadProps {
  [x: string]: any;
  changeHook?: (s: string) => string;
}

const PluginUpload: React.FC<PluginUploadProps> = (props) => {
  const intl = getIntl();

  const fileChange = (e: UploadChangeParam<UploadFile<any>>) => {
    if (e.fileList.length > 0) {
      fileToBase64(e.fileList[0].originFileObj)
        .then((f) => {
          props.onChange(props.changeHook?.(f) ?? f);
        })
        .catch((e) => message.error(`Error: ${e.message}`));
    }
  };

  const fileRemove = () => {
    props.onChange(props.changeHook?.('') ?? '');
  };

  return (
    <Upload
      onChange={fileChange}
      onRemove={fileRemove}
      maxCount={1}
      listType="text"
      accept=".zip"
      beforeUpload={(file) => {
        if ('application/zip' !== file.type) {
          message.error(intl.get('pages.plugin.form.file.invalid'));
          return Upload.LIST_IGNORE;
        }
        return false;
      }}
    >
      <Button icon={<UploadOutlined />}>
        {intl.get('pages.plugin.form.file.upload')}
      </Button>
    </Upload>
  );
};

export default PluginUpload;
