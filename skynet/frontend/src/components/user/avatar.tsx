import { fileToBase64, getIntl } from '@/utils';
import { UploadOutlined } from '@ant-design/icons';
import { Button, Upload, message } from 'antd';
import { UploadChangeParam, UploadFile } from 'antd/es/upload';

export interface AvatarUploadProps {
  value?: string;
  [x: string]: any;
  changeHook?: (s: string) => string;
}

const AvatarUpload: React.FC<AvatarUploadProps> = (props) => {
  const intl = getIntl();

  const imgChange = (e: UploadChangeParam<UploadFile<any>>) => {
    if (e.fileList.length > 0) {
      fileToBase64(e.fileList[0].originFileObj)
        .then((f) => {
          props.onChange(props.changeHook?.(f) ?? f);
        })
        .catch((e) => message.error(`Error: ${e.message}`));
    }
  };

  const imgRemove = () => {
    props.onChange(props.changeHook?.('') ?? '');
  };

  const list: Array<UploadFile<any>> = props.value
    ? [
        {
          uid: '-1',
          name: 'avatar',
          status: 'done',
          thumbUrl: props.value,
        },
      ]
    : [];

  return (
    <Upload
      defaultFileList={list}
      onChange={imgChange}
      onRemove={imgRemove}
      maxCount={1}
      listType="picture"
      accept=".png,.jpg,.jpeg,.webp"
      beforeUpload={(file) => {
        if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
          message.error(
            intl.get('pages.user.form.avatar.invalid', { file: file.name }),
          );
          return Upload.LIST_IGNORE;
        }
        if (file.size > 1024 * 1024) {
          message.error(intl.get('app.filesize', { size: '1MB' }));
          return Upload.LIST_IGNORE;
        }
        return false;
      }}
    >
      <Button icon={<UploadOutlined />}>
        {intl.get('pages.user.form.avatar.upload')}
      </Button>
    </Upload>
  );
};

export default AvatarUpload;
