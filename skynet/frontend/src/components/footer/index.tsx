import { getIntl } from '@/utils';
import { DefaultFooter } from '@ant-design/pro-layout';

export default () => {
  const intl = getIntl();
  const currentYear = new Date().getFullYear();

  return (
    <DefaultFooter
      copyright={`${currentYear} ${intl.get('app.copyright.author')}`}
      links={[]}
    />
  );
};
