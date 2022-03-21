import { ColSpanType } from '@ant-design/pro-card/lib/type';
import { Col, Row } from 'antd';

interface RowItemProps {
  span: ColSpanType;
  lastSpan?: ColSpanType;
  text: React.ReactNode;
  item: React.ReactNode;
  nospace?: boolean;
}
const RowItem: React.FC<RowItemProps> = (props) => {
  return (
    <Row style={props.nospace ? {} : { marginBottom: 24 }} align="middle">
      <Col span={props.span}>{props.text}</Col>
      <Col span={props.lastSpan}>{props.item}</Col>
    </Row>
  );
};

export default RowItem;
